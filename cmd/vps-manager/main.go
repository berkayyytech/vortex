package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"main/internal/pages"
	"main/internal/pages/dashboard"
	"main/internal/pages/docker"
	"main/internal/pages/files"
	"main/internal/pages/logs"
	"main/internal/pages/apps"
	"main/internal/pages/backup"
	"main/internal/pages/servers"
	"main/internal/pages/services"
	"main/internal/pages/security"
	"main/internal/pages/settings"
	"main/internal/pages/terminal"

	"time"
	"os/exec"

	"main/internal/agent"
	"main/internal/components"
	"main/internal/config"
	sysengine "main/internal/engine/system"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type Router struct {
	pages       []pages.Page
	sidebarIdx  int
	activeIdx   int
	sshClient   *sshlib.Client
	sysEngine   *sysengine.Engine
	activeHost string
	activeUser string
	activePort string
	toast      components.Toast
	palette    components.Palette

	width  int
	height int
}

type switchTabMsg struct{ idx int }

func initialModel() Router {
	r := Router{
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
		},
		sidebarIdx: 0,
		activeIdx:  0,
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
		Name:        "Go to Apps",
		Description: "Switch to the Application Manager view",
		Action: func() tea.Cmd {
			return func() tea.Msg { return switchTabMsg{idx: 2} }
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
				// deployment failed but we still init engines
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

		// Queue first payload fetch
		cmds = append(cmds, r.sysEngine.FetchPayload(r.activeIdx == 5))

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
		// If payload fails, wait a bit longer then try again
		r.toast = components.NewToast("System Telemetry Disconnected", "error", 3*time.Second)
		cmds = append(cmds, sysengine.Tick(10*time.Second))

	case agent.TickMsg:
		if r.sysEngine != nil {
			cmds = append(cmds, r.sysEngine.FetchPayload(r.activeIdx == 5))
		}

	case terminal.OpenShellMsg:
		if r.activeHost != "" {
			c := exec.Command("ssh", "-p", r.activePort, r.activeUser+"@"+r.activeHost)
			return r, tea.ExecProcess(c, func(err error) tea.Msg {
				return nil
			})
		}

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

		switch msg.String() {
		case "ctrl+c":
			return r, tea.Quit
		case "ctrl+p":
			r.palette.Active = !r.palette.Active
			r.palette.Update(msg)
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
			if r.activeIdx != r.sidebarIdx {
				r.activeIdx = r.sidebarIdx
				return r, nil
			}
		}

		// Strictly pass all other KeyMsgs to the active page only
		updatedModel, cmd := r.pages[r.activeIdx].Update(msg)
		r.pages[r.activeIdx] = updatedModel.(pages.Page)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return r, tea.Batch(cmds...)
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
	accentColor := theme.Current.Accent
	primaryColor := theme.Current.Primary
	dimColor := theme.Current.Dim

	sidebarStyle := lipgloss.NewStyle().
		Width(26).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder(), false, true, false, false).
		BorderForeground(dimColor)

	contentStyle := lipgloss.NewStyle().
		Padding(1, 4)

	logo := `
 █  █ █▀▀█ █▀▀█
 █  █ █  █ █▄▄▀
  ▀▀  ▀▀▀▀ ▀ ▀▀
 V O R T E X
`
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentColor).
		Render(logo)

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
		label := fmt.Sprintf("%s  %s", p.Icon(), p.Title())
		
		var renderedItem string
		if i == r.sidebarIdx && i == r.activeIdx {
			renderedItem = selectedStyle.Render("▌ "+label)
		} else if i == r.sidebarIdx {
			// Hovering but not active
			renderedItem = lipgloss.NewStyle().Foreground(theme.Current.Text).Background(lipgloss.Color("238")).Width(22).PaddingLeft(1).Render("  " + label)
		} else if i == r.activeIdx {
			// Active but not hovering
			renderedItem = lipgloss.NewStyle().Foreground(primaryColor).Width(22).PaddingLeft(1).Render("▌ " + label)
		} else {
			renderedItem = normalStyle.Render("  "+label)
		}
		items = append(items, renderedItem)
	}

	menu := lipgloss.JoinVertical(lipgloss.Left, items...)
	sidebar := sidebarStyle.Render(menu)

	// Render the active page
	activePage := r.pages[r.activeIdx]
	
	header := lipgloss.NewStyle().
		Bold(true).
		Underline(true).
		Foreground(primaryColor).
		Render(activePage.Title())

	var pageView string
	if r.sshClient == nil && r.activeIdx != 0 {
		pageView = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("❌ Please connect to a server in the 'Servers' tab first.")
	} else {
		pageView = activePage.View()
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

	// Overlay Command Palette if active
	if r.palette.Active {
		layout = lipgloss.Place(r.width, r.height, lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Center, layout, "\n", r.palette.View()),
		)
	}

	return lipgloss.Place(r.width, r.height, lipgloss.Center, lipgloss.Center, layout)
}

func main() {
	config.InitDefaults()
	config.LoadSettings()

	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
