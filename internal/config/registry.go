package config

import (
	"fmt"
)

type SettingType string

const (
	TypeBool   SettingType = "bool"
	TypeString SettingType = "string"
	TypeInt    SettingType = "int"
	TypeSelect SettingType = "select"
)

type Setting struct {
	ID          string
	Name        string
	Category    string
	Description string
	Type        SettingType
	Value       interface{}
	Options     []string
}

var Registry []Setting
var CurrentConfig Config

func InitSettings() error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	CurrentConfig = cfg

	// Build Registry from CurrentConfig
	Registry = []Setting{
		// General
		{ID: "general.confirm_destructive", Name: "Confirm Destructive Actions", Category: "General", Description: "Confirm before killing processes or deleting files", Type: TypeBool, Value: cfg.General.ConfirmDestructive},
		{ID: "general.bulk_confirm", Name: "Bulk Action Confirmation", Category: "General", Description: "When to confirm bulk actions", Type: TypeSelect, Value: cfg.General.BulkConfirm, Options: []string{"always", "3+", "never"}},
		{ID: "general.update_check", Name: "Update Check Frequency", Category: "General", Description: "How often to check for updates", Type: TypeSelect, Value: cfg.General.UpdateCheck, Options: []string{"startup", "daily", "weekly", "never"}},
		{ID: "general.landing_view", Name: "Default Landing View", Category: "General", Description: "Initial page shown upon connecting", Type: TypeSelect, Value: cfg.General.LandingView, Options: []string{"Mission Control", "Terminal", "Servers"}},

		// Appearance
		{ID: "appearance.theme", Name: "Color Theme", Category: "Appearance", Description: "Visual theme of the application", Type: TypeSelect, Value: cfg.Appearance.Theme, Options: []string{"Tokyo Night", "Catppuccin", "Nord", "Gruvbox", "Dracula", "GitHub Dark"}},
		{ID: "appearance.wallpaper", Name: "Terminal Wallpaper", Category: "Appearance", Description: "Background ambient animation", Type: TypeSelect, Value: cfg.Appearance.Wallpaper, Options: []string{"Stars", "Mountains", "Grid", "Matrix", "Minimal Dots", "None"}},
		{ID: "appearance.anim_intensity", Name: "Animation Intensity", Category: "Appearance", Description: "Disable animations for performance", Type: TypeSelect, Value: cfg.Appearance.AnimationIntensity, Options: []string{"Full", "Reduced", "Off"}},
		{ID: "appearance.graph_style", Name: "Graph Style", Category: "Appearance", Description: "Visual style for metrics", Type: TypeSelect, Value: cfg.Appearance.GraphStyle, Options: []string{"Sparkline", "Bar", "Plain"}},
		{ID: "appearance.border_style", Name: "Border Style", Category: "Appearance", Description: "Window borders", Type: TypeSelect, Value: cfg.Appearance.BorderStyle, Options: []string{"rounded", "square"}},

		// SSH
		{ID: "ssh.timeout", Name: "Connection Timeout (s)", Category: "SSH", Description: "Timeout for connecting", Type: TypeInt, Value: cfg.SSH.Timeout},
		{ID: "ssh.keepalive", Name: "Keepalive Interval (s)", Category: "SSH", Description: "Interval to send keep-alives", Type: TypeInt, Value: cfg.SSH.Keepalive},
		{ID: "ssh.auth_order", Name: "Auth Method Ordering", Category: "SSH", Description: "Order of SSH authentication methods", Type: TypeSelect, Value: cfg.SSH.AuthOrder, Options: []string{"key,agent,password", "password,key,agent"}},
		{ID: "ssh.known_hosts", Name: "Known Hosts Strictness", Category: "SSH", Description: "Strictness of host verification", Type: TypeSelect, Value: cfg.SSH.KnownHosts, Options: []string{"strict", "ask"}},

		// Monitoring
		{ID: "monitoring.refresh", Name: "Live Refresh Interval (s)", Category: "Monitoring", Description: "Telemetry polling interval", Type: TypeInt, Value: cfg.Monitoring.RefreshInterval},
		{ID: "monitoring.cpu_warn", Name: "CPU Warning Threshold %", Category: "Monitoring", Description: "Percentage to show yellow warning", Type: TypeInt, Value: cfg.Monitoring.CPUWarning},
		{ID: "monitoring.cpu_crit", Name: "CPU Critical Threshold %", Category: "Monitoring", Description: "Percentage to show red critical", Type: TypeInt, Value: cfg.Monitoring.CPUCritical},
		{ID: "monitoring.retention", Name: "Data Retention (Hours)", Category: "Monitoring", Description: "History kept for sparklines", Type: TypeInt, Value: cfg.Monitoring.DataRetentionHours},

		// Docker
		{ID: "docker.refresh", Name: "Container Refresh Interval (s)", Category: "Docker", Description: "Refresh rate for containers list", Type: TypeInt, Value: cfg.Docker.AutoRefreshInterval},
		{ID: "docker.log_tail", Name: "Default Log Tail Length", Category: "Docker", Description: "Lines loaded on open", Type: TypeInt, Value: cfg.Docker.DefaultLogTail},
		{ID: "docker.confirm", Name: "Confirm Docker Actions", Category: "Docker", Description: "Confirm container restart/stop", Type: TypeBool, Value: cfg.Docker.ConfirmActions},

		// Logs
		{ID: "logs.default_source", Name: "Default Log Source", Category: "Logs", Description: "Initial logs loaded", Type: TypeSelect, Value: cfg.Logs.DefaultSource, Options: []string{"System (journalctl)", "Syslog (/var/log/syslog)"}},
		{ID: "logs.max_buffer", Name: "Max In-Memory Line Buffer", Category: "Logs", Description: "Maximum lines kept in RAM", Type: TypeInt, Value: cfg.Logs.MaxLineBuffer},

		// Security
		{ID: "security.fail_threshold", Name: "Failed Login Alert Threshold", Category: "Security", Description: "Notify after N failed SSH logins", Type: TypeInt, Value: cfg.Security.FailAlertThreshold},
		{ID: "security.fail2ban", Name: "Fail2ban Integration", Category: "Security", Description: "Show fail2ban stats if installed", Type: TypeBool, Value: cfg.Security.Fail2BanToggle},
		{ID: "security.idle_timeout", Name: "Idle Session Timeout (s)", Category: "Security", Description: "Auto-lock inactive session", Type: TypeInt, Value: cfg.Security.IdleTimeout},

		// Notifications
		{ID: "notif.master", Name: "Master Notifications Toggle", Category: "Notifications", Description: "Enable/disable all popup toasts", Type: TypeBool, Value: cfg.Notifications.MasterToggle},
		{ID: "notif.health", Name: "Health Warnings", Category: "Notifications", Description: "Notify when resource thresholds exceeded", Type: TypeBool, Value: cfg.Notifications.HealthWarnings},
		{ID: "notif.cert", Name: "Cert Expiry Warnings", Category: "Notifications", Description: "Notify when SSL certs expire soon", Type: TypeBool, Value: cfg.Notifications.CertExpiry},
		{ID: "notif.duration", Name: "Notification Duration (s)", Category: "Notifications", Description: "How long notifications stay on screen", Type: TypeInt, Value: cfg.Notifications.DurationSeconds},

		// Backups
		{ID: "backups.dest_path", Name: "Default Backup Target Path", Category: "Backups", Description: "Where backups are stored remotely", Type: TypeString, Value: cfg.Backups.DestPath},
		{ID: "backups.schedule", Name: "Backup Schedule", Category: "Backups", Description: "Cron format for auto-backups", Type: TypeString, Value: cfg.Backups.Schedule},
		{ID: "backups.retention", Name: "Backup Retention (Count)", Category: "Backups", Description: "Number of backups to keep", Type: TypeInt, Value: cfg.Backups.Retention},
	}
	
	// Add Keybinds dynamically
	for name, key := range cfg.Keybinds {
		Registry = append(Registry, Setting{
			ID: "keybind." + name,
			Name: "Keybind: " + name,
			Category: "Keybinds",
			Description: "Press Enter to remap",
			Type: TypeString,
			Value: key,
		})
	}
	
	return nil
}

func SaveSettings() error {
	// Map Registry back to CurrentConfig
	for _, s := range Registry {
		switch s.ID {
		case "general.confirm_destructive": CurrentConfig.General.ConfirmDestructive = s.Value.(bool)
		case "general.bulk_confirm": CurrentConfig.General.BulkConfirm = s.Value.(string)
		case "general.update_check": CurrentConfig.General.UpdateCheck = s.Value.(string)
		case "general.landing_view": CurrentConfig.General.LandingView = s.Value.(string)
		
		case "appearance.theme": CurrentConfig.Appearance.Theme = s.Value.(string)
		case "appearance.wallpaper": CurrentConfig.Appearance.Wallpaper = s.Value.(string)
		case "appearance.anim_intensity": CurrentConfig.Appearance.AnimationIntensity = s.Value.(string)
		case "appearance.graph_style": CurrentConfig.Appearance.GraphStyle = s.Value.(string)
		case "appearance.border_style": CurrentConfig.Appearance.BorderStyle = s.Value.(string)
		
		case "ssh.timeout": CurrentConfig.SSH.Timeout = s.Value.(int)
		case "ssh.keepalive": CurrentConfig.SSH.Keepalive = s.Value.(int)
		case "ssh.auth_order": CurrentConfig.SSH.AuthOrder = s.Value.(string)
		case "ssh.known_hosts": CurrentConfig.SSH.KnownHosts = s.Value.(string)
		
		case "monitoring.refresh": CurrentConfig.Monitoring.RefreshInterval = s.Value.(int)
		case "monitoring.cpu_warn": CurrentConfig.Monitoring.CPUWarning = s.Value.(int)
		case "monitoring.cpu_crit": CurrentConfig.Monitoring.CPUCritical = s.Value.(int)
		case "monitoring.retention": CurrentConfig.Monitoring.DataRetentionHours = s.Value.(int)
		
		case "docker.refresh": CurrentConfig.Docker.AutoRefreshInterval = s.Value.(int)
		case "docker.log_tail": CurrentConfig.Docker.DefaultLogTail = s.Value.(int)
		case "docker.confirm": CurrentConfig.Docker.ConfirmActions = s.Value.(bool)
		
		case "logs.default_source": CurrentConfig.Logs.DefaultSource = s.Value.(string)
		case "logs.max_buffer": CurrentConfig.Logs.MaxLineBuffer = s.Value.(int)
		
		case "security.fail_threshold": CurrentConfig.Security.FailAlertThreshold = s.Value.(int)
		case "security.fail2ban": CurrentConfig.Security.Fail2BanToggle = s.Value.(bool)
		case "security.idle_timeout": CurrentConfig.Security.IdleTimeout = s.Value.(int)
		
		case "notif.master": CurrentConfig.Notifications.MasterToggle = s.Value.(bool)
		case "notif.health": CurrentConfig.Notifications.HealthWarnings = s.Value.(bool)
		case "notif.cert": CurrentConfig.Notifications.CertExpiry = s.Value.(bool)
		case "notif.duration": CurrentConfig.Notifications.DurationSeconds = s.Value.(int)
		
		case "backups.dest_path": CurrentConfig.Backups.DestPath = s.Value.(string)
		case "backups.schedule": CurrentConfig.Backups.Schedule = s.Value.(string)
		case "backups.retention": CurrentConfig.Backups.Retention = s.Value.(int)
		}
		
		if s.Category == "Keybinds" {
			name := s.ID[8:] // trim "keybind."
			CurrentConfig.Keybinds[name] = s.Value.(string)
		}
	}
	
	// Save to config.yaml
	return SaveConfig(CurrentConfig)
}

func UpdateSettingValue(id string, val interface{}) {
	for i, s := range Registry {
		if s.ID == id {
			// type cast if integer because input might be int, string, etc
			if s.Type == TypeInt {
				if v, ok := val.(string); ok {
					var intVal int
					fmt.Sscanf(v, "%d", &intVal)
					Registry[i].Value = intVal
				} else {
					Registry[i].Value = val
				}
			} else {
				Registry[i].Value = val
			}
			return
		}
	}
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
