package config

func InitDefaults() {
	// General -> Startup
	RegisterSetting(Setting{
		ID:          "startup.page",
		Name:        "Startup Page",
		Category:    "General",
		Description: "The page to open when connecting to a server",
		Type:        TypeSelect,
		Default:     "Dashboard",
		Options:     []string{"Dashboard", "Servers", "Terminal"},
	})
	RegisterSetting(Setting{
		ID:          "startup.remember_server",
		Name:        "Remember Last Server",
		Category:    "General",
		Description: "Automatically select the last connected server",
		Type:        TypeBool,
		Default:     false,
	})

	// Appearance -> Themes
	RegisterSetting(Setting{
		ID:          "appearance.theme",
		Name:        "Color Theme",
		Category:    "Appearance",
		Description: "The visual theme of the application",
		Type:        TypeSelect,
		Default:     "Vortex Dark",
		Options:     []string{"Vortex Dark", "Nord", "Dracula", "Catppuccin"},
	})
	RegisterSetting(Setting{
		ID:          "appearance.compact",
		Name:        "Compact Mode",
		Category:    "Appearance",
		Description: "Reduce padding and margins for dense information",
		Type:        TypeBool,
		Default:     false,
	})

	// SSH Settings
	RegisterSetting(Setting{
		ID:          "ssh.timeout",
		Name:        "Connection Timeout",
		Category:    "SSH",
		Description: "Timeout for SSH connections (seconds)",
		Type:        TypeInt,
		Default:     30,
	})
	RegisterSetting(Setting{
		ID:          "ssh.keep_alive",
		Name:        "Keep Alive",
		Category:    "SSH",
		Description: "Send periodic keep-alive packets to prevent disconnects",
		Type:        TypeBool,
		Default:     true,
	})

	// Monitoring Settings
	RegisterSetting(Setting{
		ID:          "monitoring.refresh_rate",
		Name:        "Dashboard Refresh Rate",
		Category:    "Monitoring",
		Description: "How often to poll the system payload",
		Type:        TypeSelect,
		Default:     "5s",
		Options:     []string{"1s", "2s", "5s", "10s"},
	})

	// Docker Settings
	RegisterSetting(Setting{
		ID:          "docker.show_stopped",
		Name:        "Show Stopped Containers",
		Category:    "Docker",
		Description: "Include stopped containers in the list",
		Type:        TypeBool,
		Default:     true,
	})

	// Log Settings
	RegisterSetting(Setting{
		ID:          "logs.default_lines",
		Name:        "Default Log Lines",
		Category:    "Logs",
		Description: "Number of lines to fetch initially",
		Type:        TypeSelect,
		Default:     "100",
		Options:     []string{"50", "100", "500", "1000"},
	})

	// Security Settings
	RegisterSetting(Setting{
		ID:          "security.auto_scan",
		Name:        "Automatic Security Scan",
		Category:    "Security",
		Description: "Automatically scan for vulnerabilities upon connection",
		Type:        TypeBool,
		Default:     true,
	})
}
