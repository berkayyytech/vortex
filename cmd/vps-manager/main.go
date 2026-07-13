package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/pages"
	"main/internal/pages/apps"
	"main/internal/pages/backup"
	"main/internal/pages/certs"
	"main/internal/pages/cron"
	"main/internal/pages/dashboard"
	"main/internal/pages/docker"
	"main/internal/pages/files"
	"main/internal/pages/logs"
	"main/internal/pages/security"
	"main/internal/pages/servers"
	"main/internal/pages/services"
	"main/internal/pages/settings"
	"main/internal/pages/terminal"

	"os/exec"
	"strings"
	"time"

	"main/internal/agent"
	"main/internal/components"
	"main/internal/config"
	sysengine "main/internal/engine/system"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type Router struct {
	startup    components.Startup
	globe      components.Globe
	pages      []pages.Page
	sidebarIdx  int
	activeIdx   int
	splitIdx    int
	isSplit     bool
	activeFocus int // 0 = main, 1 = split

	wallpaper   string
	wallpaperOn bool

	sshClient  *sshlib.Client
	sysEngine  *sysengine.Engine
	activeHost string
	activeUser string
	activePort string
	payload    agent.Payload
	isFetching bool
	toast      components.Toast
	palette    components.Palette

	width  int
	height int
}

type switchTabMsg struct{ idx int }
type toggleWallpaperMsg struct{}
type switchThemeMsg struct{ name string }

func initialModel() Router {
	r := Router{
		startup:   components.NewStartup(),
		globe:     components.NewGlobe(),
		pages: []pages.Page{
			servers.New(),
			dashboard.New(),
			apps.New(),
			docker.New(),
			services.New(),
			files.New(),
			logs.New(),
			security.New(),
			backup.New(),
			terminal.New(),
			settings.New(),
			cron.New(),
			certs.New(),
		},
		sidebarIdx:  0,
		activeIdx:   0,
		splitIdx:    -1,
		isSplit:     false,
		activeFocus: 0,
		wallpaper:   "· ",
		wallpaperOn: true,
	}

	r.palette = components.NewPalette()
	r.palette.RegisterCommand(components.Command{
		Name:        "Go to Dashboard",
		Description: "Switch to the dashboard view",
		Action: func() tea.Cmd {
			return func() tea.Msg { return switchTabMsg{idx: 1} }
		},
	})
	r.palette.RegisterCommand(components.Command{
		Name:        "Go to Docker",
		Description: "Switch to the Docker management view",
		Action: func() tea.Cmd {
			return func() tea.Msg { return switchTabMsg{idx: 3} }
		},
	})
	r.palette.RegisterCommand(components.Command{
		Name:        "Restart Docker",
		Description: "systemctl restart docker",
		Action: func() tea.Cmd {
			return func() tea.Msg { return pages.RunRemoteCmdMsg{Command: "systemctl restart docker"} }
		},
	})
	r.palette.RegisterCommand(components.Command{
		Name:        "Restart nginx",
		Description: "systemctl restart nginx",
		Action: func() tea.Cmd {
			return func() tea.Msg { return pages.RunRemoteCmdMsg{Command: "systemctl restart nginx"} }
		},
	})
	r.palette.RegisterCommand(components.Command{
		Name:        "Toggle Wallpaper",
		Description: "Enable or disable terminal background wallpaper",
		Action: func() tea.Cmd {
			return func() tea.Msg { return toggleWallpaperMsg{} }
		},
	})
	
	// Register Themes
	for _, t := range theme.Themes {
		tName := t.Name // capture
		r.palette.RegisterCommand(components.Command{
			Name:        "Theme: " + tName,
			Description: "Switch theme to " + tName,
			Action: func() tea.Cmd {
				return func() tea.Msg { return switchThemeMsg{name: tName} }
			},
		})
	}

	r.palette.RegisterCommand(components.Command{
		Name:        "Go to Files",
		Description: "Switch to the File Manager view",
		Action: func() tea.Cmd {
			return func() tea.Msg { return switchTabMsg{idx: 5} }
		},
	})

	r.palette.RegisterCommand(components.Command{
		Name:        "Go to Logs",
		Description: "Switch to the Logs view",
		Action: func() tea.Cmd {
			return func() tea.Msg { return switchTabMsg{idx: 6} }
		},
	})
	r.palette.RegisterCommand(components.Command{
		Name:        "Exit Vortex",
		Description: "Quit the application",
		Action: func() tea.Cmd {
			return tea.Quit
		},
	})

	return r
}

func (r Router) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, components.TickStartup())
	for _, p := range r.pages {
		if cmd := p.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (r Router) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case components.TickStartupMsg:
		if r.startup.Active {
			cmd := r.startup.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return r, tea.Batch(cmds...)
		}
	case components.TickGlobeMsg:
		if r.globe.Active {
			cmd := r.globe.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return r, tea.Batch(cmds...)
		}
	case sshlib.ConnectedMsg:
		r.sshClient = msg.Client
		r.sysEngine = sysengine.NewEngine(r.sshClient)
		r.activeHost = msg.Host
		r.activeUser = msg.User
		r.activePort = msg.Port
		r.sidebarIdx = 1 // Auto-switch to Dashboard
		r.activeIdx = 1

		// Async agent deployment
		cmds = append(cmds, func() tea.Msg {
			_, err := r.sshClient.DeployAndRunAgent()
			if err != nil {
				return agent.PayloadErrorMsg{Err: fmt.Errorf("Deploy failed: %v", err)}
			}
			return pages.EngineReadyMsg{Client: msg.Client}
		})

	case pages.EngineReadyMsg:
		// Broadcast to all pages to init engines
		for i, p := range r.pages {
			updatedModel, cmd := p.Update(msg)
			r.pages[i] = updatedModel.(pages.Page)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		if !r.isFetching {
			r.isFetching = true
			cmds = append(cmds, r.sysEngine.FetchPayload(r.activeIdx == 5))
		}

	case tea.WindowSizeMsg:
		r.width = msg.Width
		r.height = msg.Height

		// Broadcast window resize to all pages
		for i, p := range r.pages {
			updatedModel, _ := p.Update(msg)
			r.pages[i] = updatedModel.(pages.Page)
		}
		return r, nil

	case agent.Payload:
		r.isFetching = false
		r.payload = msg
		// Send the payload to all pages for rendering
		for i, p := range r.pages {
			updatedModel, cmd := p.Update(msg)
			r.pages[i] = updatedModel.(pages.Page)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		// Queue next tick when we successfully receive a payload
		cmds = append(cmds, sysengine.Tick(5*time.Second))

	case agent.PayloadErrorMsg:
		r.isFetching = false
		// If payload fails, wait a bit longer then try again
		errMsg := msg.Err.Error()
		r.toast = components.NewToast("Telemetry Error: "+errMsg, "error", 4*time.Second)
		cmds = append(cmds, sysengine.Tick(10*time.Second))

	case agent.TickMsg:
		if r.sysEngine != nil && !r.isFetching {
			r.isFetching = true
			cmds = append(cmds, r.sysEngine.FetchPayload(r.activeIdx == 5))
		}

	case terminal.OpenShellMsg:
		if r.activeHost != "" {
			c := exec.Command("ssh", "-t", "-p", r.activePort, r.activeUser+"@"+r.activeHost)
			return r, tea.ExecProcess(c, func(err error) tea.Msg {
				return terminal.ShellClosedMsg{}
			})
		}

	case docker.OpenDockerShellMsg:
		if r.activeHost != "" {
			// docker exec -it <container> sh -c "bash || sh"
			cmdStr := fmt.Sprintf("docker exec -it %s sh -c 'bash || sh'", msg.ContainerID)
			c := exec.Command("ssh", "-t", "-p", r.activePort, r.activeUser+"@"+r.activeHost, cmdStr)
			return r, tea.ExecProcess(c, func(err error) tea.Msg {
				return terminal.ShellClosedMsg{}
			})
		}

	case terminal.ShellClosedMsg:
		// When SSH finishes, switch back to Dashboard and refresh telemetry
		r.activeIdx = 1
		r.sidebarIdx = 1
		r.toast = components.NewToast("SSH Session Closed", "success", 2*time.Second)
		// Clear terminal screen and redraw
		return r, tea.Batch(cmds...)

	case pages.RunRemoteCmdMsg:
		if r.sshClient != nil {
			cmds = append(cmds, func() tea.Msg {
				r.sshClient.Run(msg.Command)
				return nil // executed silently in bg
			})
		}

	case pages.RunRemoteQueryMsg:
		if r.sshClient != nil {
			cmds = append(cmds, func() tea.Msg {
				out := r.sshClient.RunCommand(msg.Command)
				return msg.ResponseHandler(out)
			})
		}

	case switchTabMsg:
		r.sidebarIdx = msg.idx
		r.activeIdx = msg.idx

	case tea.KeyMsg:
		if r.palette.Active {
			var cmd tea.Cmd
			r.palette, cmd = r.palette.Update(msg)
			return r, cmd
		}

		if r.globe.Active {
			switch msg.String() {
			case "esc", "g", "G":
				r.globe.Active = false
				return r, nil
			default:
				cmd := r.globe.Update(msg)
				return r, cmd
			}
		}

		switch msg.String() {
		case "g", "G":
			r.globe.Active = true
			r.globe.IsEntering = true
			r.globe.EnterProgress = 0
			return r, components.TickGlobe()
		case "tab":
			if r.isSplit {
				r.isSplit = false
				r.splitIdx = -1
				r.activeFocus = 0
			} else {
				r.isSplit = true
				r.splitIdx = r.sidebarIdx
				r.activeFocus = 1
			}
			return r, nil
		case "?":
			r.toast = components.NewToast("Shortcuts: [ & ] Sidebar | Tab Split View | Ctrl+P Palette | R Restart | S Stop | Enter Connect", "info", 8*time.Second)
			return r, nil
		case "ctrl+c":
			return r, tea.Quit
		case "ctrl+p":
			r.palette.Active = !r.palette.Active
			return r, nil
		case "shift+up", "[":
			if r.sidebarIdx > 0 {
				r.sidebarIdx--
			} else {
				r.sidebarIdx = len(r.pages) - 1
			}
			return r, nil
		case "shift+down", "]":
			if r.sidebarIdx < len(r.pages)-1 {
				r.sidebarIdx++
			} else {
				r.sidebarIdx = 0
			}
			return r, nil
		case "enter":
			// If sidebar cursor is different from active page, switch to it!
			if r.isSplit && r.activeFocus == 1 {
				if r.splitIdx != r.sidebarIdx {
					r.splitIdx = r.sidebarIdx
					return r, nil
				}
			} else {
				if r.activeIdx != r.sidebarIdx {
					r.activeIdx = r.sidebarIdx
					return r, nil
				}
			}
			// If it's the same, fall through so the active page can handle Enter
		}

		// Handle global hotkeys if not in terminal (9) or input mode
		targetIdx := r.activeIdx
		if r.isSplit && r.activeFocus == 1 {
			targetIdx = r.splitIdx
		}
		
		allowGlobalKeys := targetIdx != 9
		if inputPage, ok := r.pages[targetIdx].(interface{ IsInputActive() bool }); ok {
			if inputPage.IsInputActive() {
				allowGlobalKeys = false
			}
		}

		if allowGlobalKeys {
			switch msg.String() {
			case "esc":
				if r.toast.Active {
					r.toast.Active = false
					return r, nil
				}
				r.sidebarIdx = 1
				if r.isSplit && r.activeFocus == 1 { r.splitIdx = 1 } else { r.activeIdx = 1 }
				return r, nil
			case "f":
				r.sidebarIdx = 5
				if r.isSplit && r.activeFocus == 1 { r.splitIdx = 5 } else { r.activeIdx = 5 }
				return r, nil
			case "l":
				r.sidebarIdx = 6
				if r.isSplit && r.activeFocus == 1 { r.splitIdx = 6 } else { r.activeIdx = 6 }
				return r, nil
			case "w":
				r.sidebarIdx = 7
				if r.isSplit && r.activeFocus == 1 { r.splitIdx = 7 } else { r.activeIdx = 7 }
				return r, nil
			case "x":
				r.sidebarIdx = 11
				if r.isSplit && r.activeFocus == 1 { r.splitIdx = 11 } else { r.activeIdx = 11 }
				return r, nil
			case "c":
				r.sidebarIdx = 12
				if r.isSplit && r.activeFocus == 1 { r.splitIdx = 12 } else { r.activeIdx = 12 }
				return r, nil
			}
		}

		// Strictly pass all other KeyMsgs to the active focus page only
		targetPageIdx := r.activeIdx
		if r.isSplit && r.activeFocus == 1 {
			targetPageIdx = r.splitIdx
		}
		
		updatedModel, cmd := r.pages[targetPageIdx].Update(msg)
		r.pages[targetPageIdx] = updatedModel.(pages.Page)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return r, tea.Batch(cmds...)

	case toggleWallpaperMsg:
		r.wallpaperOn = !r.wallpaperOn
		return r, nil

	case switchThemeMsg:
		theme.SetTheme(msg.name)
		return r, nil
	}

	// Broadcast non-KeyMsgs to all pages
	for i, p := range r.pages {
		// Do not double-broadcast agent.Payload as it was handled specifically above
		if _, isPayload := msg.(agent.Payload); !isPayload {
			updatedModel, cmd := p.Update(msg)
			r.pages[i] = updatedModel.(pages.Page)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return r, tea.Batch(cmds...)
}

func (r Router) View() string {
	if r.width == 0 {
		return "Initializing..."
	}

	if r.startup.Active {
		return r.startup.View(r.width, r.height)
	}
	
	if r.globe.Active {
		return r.globe.View(r.width, r.height, r.payload, r.activeHost, r.isFetching)
	}

	// Layout definitions
	accentColor := theme.Current.Accent
	primaryColor := theme.Current.Primary
	dimColor := theme.Current.Dim

	sidebarStyle := lipgloss.NewStyle().
		Width(26).
		Height(r.height - 5). // Stretch border all the way down to status bar
		Padding(1, 2).
		Border(lipgloss.RoundedBorder(), false, true, false, false).
		BorderForeground(dimColor)

	contentStyle := lipgloss.NewStyle().
		Padding(1, 4)

	logoLines := []string{
		` █  █ █▀▀█ █▀▀█`,
		` █  █ █  █ █▄▄▀`,
		`  ▀▀  ▀▀▀▀ ▀ ▀▀`,
		` V O R T E X`,
	}
	
	// Subtle text gradient for the logo
	title := ""
	for i, l := range logoLines {
		color := primaryColor
		if i == 1 { color = lipgloss.Color("81") }
		if i == 2 { color = lipgloss.Color("39") }
		if i == 3 { color = accentColor }
		title += lipgloss.NewStyle().Bold(true).Foreground(color).Render(l)
		if i < 3 { title += "\n" }
	}

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Background(theme.Current.HighlightBg).
		Width(22).
		PaddingLeft(1)

	normalStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Text).
		Width(22).
		PaddingLeft(1)

	var items []string
	items = append(items, title)
	items = append(items, "")
	items = append(items, lipgloss.NewStyle().Foreground(dimColor).Render(" MAIN MENU"))
	items = append(items, lipgloss.NewStyle().Foreground(dimColor).Render(" [ ] to navigate"))
	items = append(items, "")

	// Dynamically build the sidebar from the registered pages
	for i, p := range r.pages {
		title := p.Title()
		icon := p.Icon()
		if title == "Docker" { icon = "🐳" }
		if title == "Services" { icon = "⚙" }
		if title == "Files" { icon = "📁" }
		if title == "Security" { icon = "🔐" }
		if title == "Servers" { icon = "🖥" }
		if title == "Mission Control" { icon = "🚀" }

		label := fmt.Sprintf("%s  %s", icon, title)
		
		// Add live counts to label
		countStr := ""
		if title == "Docker" && r.payload.Docker.Containers > 0 {
			countStr = lipgloss.NewStyle().Foreground(theme.Current.Accent).Render(fmt.Sprintf("%d", r.payload.Docker.Containers))
		} else if title == "Services" && len(r.payload.Services) > 0 {
			countStr = lipgloss.NewStyle().Foreground(theme.Current.Accent).Render(fmt.Sprintf("%d", len(r.payload.Services)))
		}

		if countStr != "" {
			pad := 18 - lipgloss.Width(label)
			if pad < 1 { pad = 1 }
			label += strings.Repeat(" ", pad) + countStr
		}

		var renderedItem string
		if i == r.sidebarIdx && i == r.activeIdx {
			renderedItem = selectedStyle.Render("▌ " + label)
		} else if i == r.sidebarIdx {
			// Hovering but not active
			renderedItem = lipgloss.NewStyle().Foreground(theme.Current.Text).Background(lipgloss.Color("238")).Width(22).PaddingLeft(1).Render("  " + label)
		} else if i == r.activeIdx {
			// Active but not hovering
			renderedItem = lipgloss.NewStyle().Foreground(primaryColor).Width(22).PaddingLeft(1).Render("▌ " + label)
		} else {
			renderedItem = normalStyle.Render("  " + label)
		}
		items = append(items, renderedItem)
	}

	menu := lipgloss.JoinVertical(lipgloss.Left, items...)
	sidebar := sidebarStyle.Render(menu)

	// Render the active page
	activePage := r.pages[r.activeIdx]

	headerTitle := activePage.Title()
	if r.isSplit && r.splitIdx >= 0 {
		headerTitle += "  |  " + r.pages[r.splitIdx].Title()
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Underline(true).
		Foreground(primaryColor).
		Render(headerTitle)

	var pageView string
	if r.palette.Active {
		pageView = r.palette.View()
	} else if r.sshClient == nil && r.activeIdx != 0 {
		pageView = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("❌ Please connect to a server in the 'Servers' tab first.")
	} else {
		// Normal View
		if r.isSplit && r.splitIdx >= 0 {
			leftView := activePage.View()
			rightView := r.pages[r.splitIdx].View()
			
			// Try to divide width evenly for components if they support it, but for now just join
			pageView = lipgloss.JoinHorizontal(lipgloss.Top, leftView, "    ", rightView)
		} else {
			pageView = activePage.View()
		}
	}

	content := contentStyle.Render(
		header + "\n\n" + pageView,
	)

	// Combine Sidebar and Content horizontally
	layout := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)

	// Overlay Toast if active
	r.toast.Update()
	if r.toast.Active {
		layout = lipgloss.JoinVertical(lipgloss.Center, layout, r.toast.View())
	}

	// Status Bar
	statusBarStr := ""
	if r.activeHost != "" {
		ramStr := "0%"
		if r.payload.Stats.RAMPercent > 0 {
			ramStr = fmt.Sprintf("%.0f%%", r.payload.Stats.RAMPercent)
		}
		cpuStr := "0%"
		if r.payload.Stats.CPUPercent > 0 {
			cpuStr = fmt.Sprintf("%.0f%%", r.payload.Stats.CPUPercent)
		}
		
		isAbnormal := false
		if r.payload.Stats.CPUPercent > 90 || r.payload.Stats.RAMPercent > 90 || r.payload.Stats.DiskPercent > 90 {
			isAbnormal = true
		}
		for _, s := range r.payload.Services {
			if s.Status == "failed" {
				isAbnormal = true
				break
			}
		}

		modeStr := lipgloss.NewStyle().Foreground(theme.Current.Bg).Background(theme.Current.Success).Render("  NORMAL  ")
		if isAbnormal {
			modeStr = lipgloss.NewStyle().Foreground(theme.Current.Bg).Background(theme.Current.Warning).Bold(true).Render("  ABNORMAL  ")
		}

		statusBarStr += modeStr
		statusBarStr += lipgloss.NewStyle().Foreground(theme.Current.Text).Render(fmt.Sprintf("  %s  |  SSH <10ms  |  %s  |  CPU %s  |  RAM %s  |  Ctrl+P Search  ? Help  Tab Focus", r.activeHost, time.Now().Format("15:04"), cpuStr, ramStr))
	} else {
		statusBarStr += lipgloss.NewStyle().Foreground(theme.Current.Text).Render("  NOT CONNECTED  |  Enter=Select Server")
	}

	statusBar := lipgloss.NewStyle().
		Padding(0, 1).
		Render(statusBarStr)

	// Combine layout and status bar
	finalLayout := lipgloss.JoinVertical(lipgloss.Left, layout, "\n", statusBar)

	// Wallpaper rendering
	if r.wallpaperOn && r.wallpaper != "" && r.width > 0 && r.height > 0 {
		return lipgloss.Place(r.width, r.height, lipgloss.Center, lipgloss.Center, finalLayout, 
			lipgloss.WithWhitespaceChars(r.wallpaper), 
			lipgloss.WithWhitespaceForeground(lipgloss.Color("235")))
	}

	return lipgloss.Place(r.width, r.height, lipgloss.Center, lipgloss.Center, finalLayout)
}

func main() {
	config.InitDefaults()
	config.LoadSettings()
	theme.SetTheme(config.GetSettingString("appearance.theme"))

	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
