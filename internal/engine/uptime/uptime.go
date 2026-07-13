package uptime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type CheckType string

const (
	HTTPCheck CheckType = "http"
	PingCheck CheckType = "ping"
)

type MonitorTarget struct {
	ID           string
	Name         string
	URL          string
	Type         CheckType
	Interval     time.Duration
	Timeout      time.Duration
	ExpectStatus int

	History      []float64
	Uptime       float64
	Status       string
	TotalChecks  int
	FailedChecks int

	Webhooks []string
}

var Targets = []*MonitorTarget{}

// EventChannel allows UI to consume events and broadcast them as LogActivityMsg
var EventChannel = make(chan string, 100)

func AddTarget(t *MonitorTarget) {
	if t.Interval == 0 {
		t.Interval = 10 * time.Second
	}
	if t.Timeout == 0 {
		t.Timeout = 5 * time.Second
	}
	if t.ExpectStatus == 0 && t.Type == HTTPCheck {
		t.ExpectStatus = 200
	}
	if t.History == nil {
		t.History = make([]float64, 0)
	}
	t.Status = "pending"
	Targets = append(Targets, t)
	
	go runCheck(t)
	go startMonitor(t)
}

func startMonitor(t *MonitorTarget) {
	ticker := time.NewTicker(t.Interval)
	for range ticker.C {
		runCheck(t)
	}
}

func runCheck(t *MonitorTarget) {
	start := time.Now()
	var err error
	var success bool

	if t.Type == HTTPCheck {
		client := http.Client{Timeout: t.Timeout}
		resp, reqErr := client.Get(t.URL)
		if reqErr != nil {
			err = reqErr
			success = false
		} else {
			resp.Body.Close()
			if resp.StatusCode == t.ExpectStatus {
				success = true
			} else {
				err = fmt.Errorf("expected %d got %d", t.ExpectStatus, resp.StatusCode)
				success = false
			}
		}
	} else if t.Type == PingCheck {
		client := http.Client{Timeout: t.Timeout}
		resp, reqErr := client.Head(t.URL)
		if reqErr != nil {
			err = reqErr
			success = false
		} else {
			resp.Body.Close()
			success = true
		}
	}

	durationMs := float64(time.Since(start).Milliseconds())
	t.TotalChecks++
	
	oldStatus := t.Status
	
	if !success {
		t.FailedChecks++
		t.Status = "down"
		t.History = append(t.History, float64(t.Timeout.Milliseconds()))
	} else {
		t.Status = "up"
		t.History = append(t.History, durationMs)
	}

	if len(t.History) > 30 {
		t.History = t.History[1:]
	}

	t.Uptime = (float64(t.TotalChecks - t.FailedChecks) / float64(t.TotalChecks)) * 100.0

	if oldStatus != "pending" && oldStatus != "" && oldStatus != t.Status {
		msg := fmt.Sprintf("Alert: Target %s is now %s. Details: %v", t.Name, t.Status, err)
		triggerWebhooks(t, msg)
		select {
		case EventChannel <- msg:
		default:
			// channel full, drop event
		}
	}
}

func triggerWebhooks(t *MonitorTarget, message string) {
	payload := map[string]string{"text": message}
	data, _ := json.Marshal(payload)
	for _, w := range t.Webhooks {
		go http.Post(w, "application/json", bytes.NewBuffer(data))
	}
}
