package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"main/internal/config"
	"main/internal/stats"
)

type WebhookType string

const (
	Discord WebhookType = "discord"
	Slack   WebhookType = "slack"
)

type Webhook struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Type      WebhookType `json:"type"`
	URL       string      `json:"url"`
	Enabled   bool        `json:"enabled"`
	LastError string      `json:"last_error"`
	LastFired time.Time   `json:"last_fired"`
}

type Engine struct {
	mu           sync.Mutex
	Webhooks     []Webhook
	filepath     string
	rateLimits   map[string]time.Time
	limitMinutes float64
}

func NewEngine() *Engine {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	path := filepath.Join(configDir, "vortex", "webhooks.json")

	e := &Engine{
		filepath:     path,
		Webhooks:     []Webhook{},
		rateLimits:   make(map[string]time.Time),
		limitMinutes: 5.0, // 5 minute rate limit per alert type
	}
	e.Load()
	return e
}

func (e *Engine) Load() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	data, err := os.ReadFile(e.filepath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &e.Webhooks)
}

func (e *Engine) Save() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	os.MkdirAll(filepath.Dir(e.filepath), 0755)
	data, err := json.MarshalIndent(e.Webhooks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(e.filepath, data, 0644)
}

func (e *Engine) AddWebhook(w Webhook) error {
	w.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	w.LastError = "Never fired"
	e.mu.Lock()
	e.Webhooks = append(e.Webhooks, w)
	e.mu.Unlock()
	return e.Save()
}

func (e *Engine) DeleteWebhook(id string) error {
	e.mu.Lock()
	for i, w := range e.Webhooks {
		if w.ID == id {
			e.Webhooks = append(e.Webhooks[:i], e.Webhooks[i+1:]...)
			break
		}
	}
	e.mu.Unlock()
	return e.Save()
}

func (e *Engine) ToggleWebhook(id string) error {
	e.mu.Lock()
	for i, w := range e.Webhooks {
		if w.ID == id {
			e.Webhooks[i].Enabled = !e.Webhooks[i].Enabled
			break
		}
	}
	e.mu.Unlock()
	return e.Save()
}

func (e *Engine) GetWebhooks() []Webhook {
	e.mu.Lock()
	defer e.mu.Unlock()
	res := make([]Webhook, len(e.Webhooks))
	copy(res, e.Webhooks)
	return res
}

// CheckThresholds compares stats against config.yaml thresholds
func (e *Engine) CheckThresholds(sys stats.SystemStats, cfg config.MonitoringConfig) {
	if sys.CPUPercent >= float64(cfg.CPUCritical) {
		e.TriggerAlert("CPU_CRITICAL", fmt.Sprintf("CPU usage is critical: %.2f%%", sys.CPUPercent))
	} else if sys.CPUPercent >= float64(cfg.CPUWarning) {
		e.TriggerAlert("CPU_WARNING", fmt.Sprintf("CPU usage is warning: %.2f%%", sys.CPUPercent))
	}

	if sys.RAMPercent >= float64(cfg.RAMCritical) {
		e.TriggerAlert("RAM_CRITICAL", fmt.Sprintf("RAM usage is critical: %.2f%%", sys.RAMPercent))
	} else if sys.RAMPercent >= float64(cfg.RAMWarning) {
		e.TriggerAlert("RAM_WARNING", fmt.Sprintf("RAM usage is warning: %.2f%%", sys.RAMPercent))
	}

	if sys.DiskPercent >= float64(cfg.DiskCritical) {
		e.TriggerAlert("DISK_CRITICAL", fmt.Sprintf("Disk usage is critical: %.2f%%", sys.DiskPercent))
	} else if sys.DiskPercent >= float64(cfg.DiskWarning) {
		e.TriggerAlert("DISK_WARNING", fmt.Sprintf("Disk usage is warning: %.2f%%", sys.DiskPercent))
	}
}

func (e *Engine) TriggerAlert(alertKey, message string) {
	e.mu.Lock()
	lastTime, exists := e.rateLimits[alertKey]
	if exists && time.Since(lastTime).Minutes() < e.limitMinutes {
		e.mu.Unlock()
		return // Rate limited
	}
	e.rateLimits[alertKey] = time.Now()
	webhooks := append([]Webhook{}, e.Webhooks...)
	e.mu.Unlock()

	go func() {
		for _, w := range webhooks {
			if !w.Enabled {
				continue
			}
			
			err := e.sendPayload(w, message)
			e.mu.Lock()
			for j, ew := range e.Webhooks {
				if ew.ID == w.ID {
					e.Webhooks[j].LastFired = time.Now()
					if err != nil {
						e.Webhooks[j].LastError = err.Error()
					} else {
						e.Webhooks[j].LastError = "Delivered successfully"
					}
				}
			}
			e.mu.Unlock()
		}
		e.Save()
	}()
}

func (e *Engine) sendPayload(w Webhook, message string) error {
	var payload []byte
	var err error

	if w.Type == Discord {
		p := map[string]string{"content": "[Vortex Alert] " + message}
		payload, err = json.Marshal(p)
	} else if w.Type == Slack {
		p := map[string]string{"text": "[Vortex Alert] " + message}
		payload, err = json.Marshal(p)
	} else {
		return fmt.Errorf("unknown webhook type")
	}

	if err != nil {
		return err
	}

	resp, err := http.Post(w.URL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
