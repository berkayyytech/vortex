package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type SettingType string

const (
	TypeBool   SettingType = "bool"
	TypeString SettingType = "string"
	TypeInt    SettingType = "int"
	TypeSelect SettingType = "select"
)

type Setting struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Category    string      `json:"category"`
	Description string      `json:"description"`
	Type        SettingType `json:"type"`
	Default     interface{} `json:"default"`
	Value       interface{} `json:"value"`
	Options     []string    `json:"options,omitempty"`
}

var Registry []Setting

func RegisterSetting(s Setting) {
	if s.Value == nil {
		s.Value = s.Default
	}
	Registry = append(Registry, s)
}

func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "vortex")
	os.MkdirAll(dir, 0755)
	return dir, nil
}

func LoadSettings() error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "settings.json")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return SaveSettings() // Create with defaults
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Load stored values into a map
	var stored []Setting
	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}

	valMap := make(map[string]interface{})
	for _, s := range stored {
		valMap[s.ID] = s.Value
	}

	// Apply to registry
	for i, reg := range Registry {
		if val, exists := valMap[reg.ID]; exists {
			Registry[i].Value = val
		}
	}

	return nil
}

func SaveSettings() error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "settings.json")

	data, err := json.MarshalIndent(Registry, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func GetSettingValue(id string) interface{} {
	for _, s := range Registry {
		if s.ID == id {
			return s.Value
		}
	}
	return nil
}

func GetSettingBool(id string) bool {
	val := GetSettingValue(id)
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}

func GetSettingString(id string) string {
	val := GetSettingValue(id)
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

// ApplySettings synchronizes the configuration state with the application subsystems (e.g., themes)
func ApplySettings() {
	// Not ideal to import theme here due to circular deps if we aren't careful, 
	// but it's safe if theme doesn't import config.
	// Actually, wait, let's just do it in main.go to avoid circular deps.
}
