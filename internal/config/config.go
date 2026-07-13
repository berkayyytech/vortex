package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Name     string `yaml:"name" json:"name"`
	Host     string `yaml:"host" json:"host"`
	Port     string `yaml:"port" json:"port"`
	User     string `yaml:"user" json:"user"`
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
	KeyPath  string `yaml:"key_path,omitempty" json:"key_path,omitempty"`
}

type GeneralConfig struct {
	ConfirmDestructive bool   `yaml:"confirm_destructive"`
	BulkConfirm        string `yaml:"bulk_confirm"` // always, 3+, never
	UpdateCheck        string `yaml:"update_check"` // startup, daily, weekly, never
	LandingView        string `yaml:"landing_view"`
}

type AppearanceConfig struct {
	Theme             string `yaml:"theme"` // Tokyo Night, Catppuccin, etc.
	Wallpaper         string `yaml:"wallpaper"`
	WallpaperOpacity  int    `yaml:"wallpaper_opacity"`
	AnimationIntensity string `yaml:"animation_intensity"` // Full, Reduced, Off
	GraphStyle        string `yaml:"graph_style"` // Sparkline, Bar, Plain
	BorderStyle       string `yaml:"border_style"`
}

type SSHConfig struct {
	Timeout    int    `yaml:"timeout_seconds"`
	Keepalive  int    `yaml:"keepalive_seconds"`
	AuthOrder  string `yaml:"auth_order"`
	KnownHosts string `yaml:"known_hosts"` // strict, ask, off
}

type MonitoringConfig struct {
	CPUWarning         int `yaml:"cpu_warning"`
	CPUCritical        int `yaml:"cpu_critical"`
	RAMWarning         int `yaml:"ram_warning"`
	RAMCritical        int `yaml:"ram_critical"`
	DiskWarning        int `yaml:"disk_warning"`
	DiskCritical       int `yaml:"disk_critical"`
	RefreshInterval    int `yaml:"refresh_interval"`
	DataRetentionHours int `yaml:"data_retention_hours"`
}

type DockerConfig struct {
	AutoRefreshInterval int  `yaml:"auto_refresh_interval"`
	DefaultLogTail      int  `yaml:"default_log_tail"`
	ConfirmActions      bool `yaml:"confirm_actions"`
}

type LogsConfig struct {
	DefaultSource string `yaml:"default_source"`
	MaxLineBuffer int    `yaml:"max_line_buffer"`
}

type SecurityConfig struct {
	FailAlertThreshold int  `yaml:"fail_alert_threshold"`
	Fail2BanToggle     bool `yaml:"fail2ban_toggle"`
	IdleTimeout        int  `yaml:"idle_timeout"`
}

type NotificationsConfig struct {
	MasterToggle     bool `yaml:"master_toggle"`
	DockerEvents     bool `yaml:"docker_events"`
	ServiceFails     bool `yaml:"service_fails"`
	HealthWarnings   bool `yaml:"health_warnings"`
	SSHConnects      bool `yaml:"ssh_connects"`
	UpdatesAvailable bool `yaml:"updates_available"`
	CertExpiry       bool `yaml:"cert_expiry"`
	BulkComplete     bool `yaml:"bulk_complete"`
	DurationSeconds  int  `yaml:"duration_seconds"`
	Sound            bool `yaml:"sound"`
}

type BackupsConfig struct {
	DestPath  string `yaml:"dest_path"`
	Schedule  string `yaml:"schedule"`
	Retention int    `yaml:"retention"`
}

type KeybindsConfig map[string]string

type UptimeMonitorConfig struct {
	ID           string `yaml:"id"`
	Name         string `yaml:"name"`
	URL          string `yaml:"url"`
	Type         string `yaml:"type"` // "http" or "ping"
	IntervalSecs int    `yaml:"interval_secs"`
}

type Config struct {
	Servers       []ServerConfig          `yaml:"servers"`
	General       GeneralConfig           `yaml:"general"`
	Appearance    AppearanceConfig        `yaml:"appearance"`
	SSH           SSHConfig               `yaml:"ssh"`
	Monitoring    MonitoringConfig        `yaml:"monitoring"`
	Docker        DockerConfig            `yaml:"docker"`
	Logs          LogsConfig              `yaml:"logs"`
	Security      SecurityConfig          `yaml:"security"`
	Notifications NotificationsConfig     `yaml:"notifications"`
	Backups       BackupsConfig           `yaml:"backups"`
	Keybinds      KeybindsConfig          `yaml:"keybinds"`
	UptimeTargets []UptimeMonitorConfig   `yaml:"uptime_targets"`
}

func GetDefaultConfig() Config {
	return Config{
		Servers: []ServerConfig{
			{Name: "Production Web (Template)", Host: "10.0.0.1", Port: "22", User: "admin", KeyPath: "~/.ssh/id_rsa"},
			{Name: "Database Node (Template)", Host: "10.0.0.2", Port: "22", User: "root", Password: "securepassword"},
		},
		General: GeneralConfig{
			ConfirmDestructive: true,
			BulkConfirm:        "3+",
			UpdateCheck:        "daily",
			LandingView:        "Mission Control",
		},
		Appearance: AppearanceConfig{
			Theme:              "Tokyo Night",
			Wallpaper:          "Stars",
			WallpaperOpacity:   30,
			AnimationIntensity: "Full",
			GraphStyle:         "Sparkline",
			BorderStyle:        "rounded",
		},
		SSH: SSHConfig{
			Timeout:    10,
			Keepalive:  15,
			AuthOrder:  "key,agent,password",
			KnownHosts: "ask",
		},
		Monitoring: MonitoringConfig{
			CPUWarning:         80,
			CPUCritical:        95,
			RAMWarning:         80,
			RAMCritical:        95,
			DiskWarning:        80,
			DiskCritical:       95,
			RefreshInterval:    2,
			DataRetentionHours: 6,
		},
		Docker: DockerConfig{
			AutoRefreshInterval: 5,
			DefaultLogTail:      100,
			ConfirmActions:      true,
		},
		Logs: LogsConfig{
			DefaultSource: "System (journalctl)",
			MaxLineBuffer: 5000,
		},
		Security: SecurityConfig{
			FailAlertThreshold: 5,
			Fail2BanToggle:     true,
			IdleTimeout:        300,
		},
		Notifications: NotificationsConfig{
			MasterToggle:     true,
			DockerEvents:     true,
			ServiceFails:     true,
			HealthWarnings:   true,
			SSHConnects:      false,
			UpdatesAvailable: true,
			CertExpiry:       true,
			BulkComplete:     true,
			DurationSeconds:  5,
			Sound:            false,
		},
		Backups: BackupsConfig{
			DestPath:  "/var/backups/vortex",
			Schedule:  "0 2 * * *",
			Retention: 7,
		},
		Keybinds: KeybindsConfig{
			"Dashboard": "g",
			"Servers": "esc", // default back
			"Docker": "d",
			"Processes": "p",
			"Services": "s",
			"Logs": "l",
			"Files": "f",
			"Security": "w",
			"Cron": "x",
			"Certs": "c",
			"Users": "u",
			"Alerts": "a",
			"Audit": "v",
			"Database": "b",
			"Proxy": "r",
			"Secrets": "e",
			"Deploy": "y",
			"Snapshots": "h",
			"Uptime": "m",
			"Backup": "k",
			"Command Palette": "ctrl+p",
		},
		UptimeTargets: []UptimeMonitorConfig{
			{
				ID:           "1",
				Name:         "Cloudflare DNS",
				URL:          "https://1.1.1.1",
				Type:         "ping",
				IntervalSecs: 5,
			},
			{
				ID:           "2",
				Name:         "Example Web",
				URL:          "https://example.com",
				Type:         "http",
				IntervalSecs: 5,
			},
		},
	}
}

func GetConfigPath() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "vortex", "config.yaml"), nil
}

func SaveConfig(cfg Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}
	os.MkdirAll(filepath.Dir(configPath), 0755)
	
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

func LoadConfig() (Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return GetDefaultConfig(), err
	}

	// Migrate from old .vortex/config.json if yaml doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		home, _ := os.UserHomeDir()
		oldPath := filepath.Join(home, ".vortex", "config.json")
		
		def := GetDefaultConfig()
		
		if data, err := os.ReadFile(oldPath); err == nil {
			// Extract servers from old json
			type oldCfg struct {
				Servers []ServerConfig `json:"servers"`
			}
			var o oldCfg
			if err := json.Unmarshal(data, &o); err == nil && len(o.Servers) > 0 {
				def.Servers = o.Servers
			}
		}
		
		// Write the new YAML file
		SaveConfig(def)
		return def, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Warning: Could not read config file, using defaults: %v\n", err)
		return GetDefaultConfig(), nil
	}

	cfg := GetDefaultConfig() // Load defaults first so missing keys fallback gracefully
	
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		// Log warning but continue
		fmt.Printf("Warning: Config file contains invalid YAML or keys. Using fallback defaults for invalid entries. Error: %v\n", err)
	}

	return cfg, nil
}
