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
	"main/internal/pages/network"
	"main/internal/pages/servers"
	"main/internal/pages/services"
	"main/internal/pages/settings"
	"main/internal/pages/terminal"

	"encoding/json"
	"os/exec"
	"time"

	"main/internal/agent"
	sshlib "main/internal/ssh"
	"main/internal/theme"
)

type Router struct {
	pages       []pages.Page
	sidebarIdx  int
	activeIdx   int
	sshClient   *sshlib.Client
	activeHost string
	activeUser string
	activePort string

	width  int
	height int

	paletteActive bool
	paletteCursor int
	paletteItems  []string
}

func initialModel() Router {
	return Router{
		pages: []pages.Page{
			servers.New(),
			dashboard.New(),
			network.New(),
			docker.New(),
			services.New(),
			files.New(),
			logs.New(),
			terminal.New(),
			settings.New(),
		},
		sidebarIdx: 0,
		activeIdx:  0,
		paletteItems: []string{
			"Restart Docker Engine",
			"Restart Nginx",
			"Clear System Cache",
			"Reboot Server",
			"Disconnect SSH",
		},
	}
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
		r.activeHost = msg.Host
		r.activeUser = msg.User
		r.activePort = msg.Port
		r.sidebarIdx = 1 // Auto-switch to Dashboard
		r.activeIdx = 1

		// Async agent deployment
		cmds = append(cmds, func() tea.Msg {
			out, err := r.sshClient.DeployAndRunAgent()
			if err != nil {
				return fmt.Errorf("Deployment failed: %v", err)
			}
			var payload agent.Payload
			if err := json.Unmarshal(out, &payload); err != nil {
				return fmt.Errorf("JSON Parse failed: %v", err)
			}
			return payload
		})

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
		// Queue next tick when we successfully receive a payload
		cmds = append(cmds, tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
			return agentTick(t)
		}))

	case agentTick:
		if r.sshClient != nil {
			cmds = append(cmds, func() tea.Msg {
				out, err := r.sshClient.Run("/tmp/vortex-agent payload")
				if err != nil {
					return nil // Ignore momentary network drops
				}
				var payload agent.Payload
				json.Unmarshal([]byte(out), &payload)

				// Fetch live logs only if the Logs tab is active
				if r.activeIdx == 5 {
					logsOut := r.sshClient.RunCommand("journalctl -n 25 --no-pager")
					payload.Logs = logsOut
				}

				return payload
			})
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

	case tea.KeyMsg:
		if r.paletteActive {
			switch msg.String() {
			case "esc", "ctrl+p":
				r.paletteActive = false
			case "up", "k":
				if r.paletteCursor > 0 {
					r.paletteCursor--
				}
			case "down", "j":
				if r.paletteCursor < len(r.paletteItems)-1 {
					r.paletteCursor++
				}
			case "enter":
				// Placeholder: Execute selected command
				r.paletteActive = false
				if r.sshClient != nil && r.paletteCursor == 0 {
					// Example: Restart docker
					r.sshClient.RunCommand("systemctl restart docker")
				}
			}
			return r, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return r, tea.Quit
		case "ctrl+p":
			r.paletteActive = true
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
			// If they are the same, let the active page handle "enter" (e.g. connecting to a server)
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
		updatedModel, cmd := p.Update(msg)
		r.pages[i] = updatedModel.(pages.Page)
		if cmd != nil {
			cmds = append(cmds, cmd)
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

	// Render Command Palette Overlay if active
	if r.paletteActive {
		paletteBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(1, 4).
			Background(lipgloss.Color("236"))

		var pItems string
		for i, item := range r.paletteItems {
			if i == r.paletteCursor {
				pItems += lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render("▶ "+item) + "\n"
			} else {
				pItems += "  " + item + "\n"
			}
		}

		overlay := paletteBox.Render(
			lipgloss.NewStyle().Bold(true).Foreground(accentColor).Render("COMMAND PALETTE (Ctrl+P)") + "\n\n" +
				pItems,
		)

		// Place overlay roughly in the center over the layout
		layout = lipgloss.Place(r.width, r.height, lipgloss.Center, lipgloss.Center, overlay)
	} else if r.width > 0 && r.height > 0 {
		// Perfectly center the main UI layout in the terminal window
		layout = lipgloss.Place(r.width, r.height, lipgloss.Center, lipgloss.Center, layout)
	}

	return layout
}

type agentTick time.Time

func main() {
	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
