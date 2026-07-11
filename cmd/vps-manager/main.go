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
)

type Router struct {
	pages      []pages.Page
	cursor     int
	sshClient  *sshlib.Client
	activeHost string
	activeUser string

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
		cursor: 0,
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
		r.cursor = 1 // Auto-switch to Dashboard

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
				if r.cursor == 5 {
					logsOut := r.sshClient.RunCommand("journalctl -n 25 --no-pager")
					payload.Logs = logsOut
				}
				// Fetch remote files only if the Files tab is active
				if r.cursor == 4 {
					filesOut := r.sshClient.RunCommand("ls -lah --color=never /")
					payload.Files = filesOut
				}

				return payload
			})
		}

	case terminal.OpenShellMsg:
		if r.activeHost != "" {
			c := exec.Command("ssh", r.activeUser+"@"+r.activeHost)
			return r, tea.ExecProcess(c, func(err error) tea.Msg {
				return nil
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
		case "ctrl+c", "q":
			return r, tea.Quit
		case "ctrl+p":
			r.paletteActive = true
			return r, nil
		case "tab", "right", "l":
			if r.cursor < len(r.pages)-1 {
				r.cursor++
			} else {
				r.cursor = 0
			}
			return r, nil
		case "shift+tab", "left", "h":
			if r.cursor > 0 {
				r.cursor--
			} else {
				r.cursor = len(r.pages)-1
			}
			return r, nil
		}

		// Strictly pass all other KeyMsgs to the active page only
		updatedModel, cmd := r.pages[r.cursor].Update(msg)
		r.pages[r.cursor] = updatedModel.(pages.Page)
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
	accentColor := lipgloss.Color("205")
	primaryColor := lipgloss.Color("86")
	dimColor := lipgloss.Color("240")

	sidebarStyle := lipgloss.NewStyle().
		Width(24).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder(), false, true, false, false).
		BorderForeground(dimColor)

	contentStyle := lipgloss.NewStyle().
		Padding(1, 2)

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(accentColor).
		Render("VORTEX VPS")

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	var items []string
	items = append(items, title)
	items = append(items, "")

	// Dynamically build the sidebar from the registered pages
	for i, p := range r.pages {
		label := fmt.Sprintf("%s %s", p.Icon(), p.Title())
		if i == r.cursor {
			items = append(items, selectedStyle.Render("▶ "+label))
		} else {
			items = append(items, normalStyle.Render("  "+label))
		}
	}

	menu := lipgloss.JoinVertical(lipgloss.Left, items...)
	sidebar := sidebarStyle.Render(menu)

	// Render the active page
	activePage := r.pages[r.cursor]
	
	header := lipgloss.NewStyle().
		Bold(true).
		Underline(true).
		Foreground(primaryColor).
		Render(activePage.Title())

	content := contentStyle.Render(
		header + "\n" + activePage.View(),
	)

	// Render Command Palette Overlay
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

		// Place overlay roughly in the center
		content = lipgloss.Place(
			60, 20,
			lipgloss.Center, lipgloss.Center,
			overlay,
		)
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebar,
		content,
	)
}

type agentTick time.Time

func main() {
	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
