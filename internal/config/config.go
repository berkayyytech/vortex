package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ServerConfig struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password,omitempty"`
	KeyPath  string `json:"key_path,omitempty"`
}

type Config struct {
	Servers []ServerConfig `json:"servers"`
}

// LoadConfig reads the servers from ~/.vortex/config.json
func LoadConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}

	configDir := filepath.Join(home, ".vortex")
	configPath := filepath.Join(configDir, "config.json")

	// If the config doesn't exist, create a production-ready default template
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		os.MkdirAll(configDir, 0755)
		defaultCfg := Config{
			Servers: []ServerConfig{
				{Name: "Production Web (Template)", Host: "10.0.0.1", Port: "22", User: "admin", KeyPath: "~/.ssh/id_rsa"},
				{Name: "Database Node (Template)", Host: "10.0.0.2", Port: "22", User: "root", Password: "securepassword"},
			},
		}
		data, _ := json.MarshalIndent(defaultCfg, "", "  ")
		os.WriteFile(configPath, data, 0644)
		return defaultCfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
