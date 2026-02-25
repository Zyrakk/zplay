package cli

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Zyrakk/zplay/internal/config"
	"github.com/Zyrakk/zplay/internal/k8s"

	// Import games to register them
	_ "github.com/Zyrakk/zplay/internal/games/terraria"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			BorderStyle(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 2).
			Margin(1, 0)

	menuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))
)

type menuModel struct {
	choices  []string
	cursor   int
	selected string
	cfg      *config.Config
	quitting bool
}

func initialMenuModel(cfg *config.Config) menuModel {
	return menuModel{
		choices: []string{
			"Deploy server",
			"List servers",
			"Start/Stop server",
			"Server status",
			"Backup server",
			"Restore backup",
			"Delete server",
			"Server console",
			"View logs",
			"Exit",
		},
		cfg: cfg,
	}
}

func (m menuModel) Init() tea.Cmd { return nil }

func (m menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			m.selected = m.choices[m.cursor]
			return m, tea.Quit
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m menuModel) View() string {
	s := titleStyle.Render("ZPlay - Game Server Manager") + "\n\n"

	for i, choice := range m.choices {
		cursor := "  "
		style := menuItemStyle

		if m.cursor == i {
			cursor = "▸ "
			style = selectedStyle
		}

		s += style.Render(cursor+choice) + "\n"
	}

	s += "\n" + dimStyle.Render("↑/↓ navigate • enter select • q quit") + "\n"
	return s
}

func Run(cfg *config.Config) error {
	// Check kubernetes connection
	client := k8s.NewClient(cfg.Kubeconfig)
	if !client.IsConnected() {
		fmt.Println(errorStyle.Render("✗ Cannot connect to Kubernetes cluster"))
		fmt.Println(dimStyle.Render("  Make sure you're logged in with: zcloud login"))
		return fmt.Errorf("kubernetes not connected")
	}

	for {
		p := tea.NewProgram(initialMenuModel(cfg))
		m, err := p.Run()
		if err != nil {
			return err
		}

		model := m.(menuModel)
		if model.quitting {
			return nil
		}

		resetTerminal()

		var actionErr error
		switch model.selected {
		case "Deploy server":
			actionErr = RunDeploy(cfg)
		case "List servers":
			actionErr = RunList(cfg)
		case "Start/Stop server":
			actionErr = RunStartStop(cfg)
		case "Server status":
			actionErr = RunStatus(cfg, "")
		case "Backup server":
			actionErr = RunBackup(cfg)
		case "Restore backup":
			actionErr = RunRestore(cfg)
		case "Delete server":
			actionErr = RunDelete(cfg)
		case "Server console":
			actionErr = RunConsole(cfg)
		case "View logs":
			actionErr = RunLogs(cfg)
		case "Exit":
			return nil
		}

		if actionErr != nil {
			fmt.Println(errorStyle.Render("Error: " + actionErr.Error()))
		}

		fmt.Println()
		fmt.Print(dimStyle.Render("Press Enter to continue..."))
		fmt.Scanln()
	}
}

func resetTerminal() {
	cmd := exec.Command("stty", "sane")
	cmd.Stdin = os.Stdin
	_ = cmd.Run()
}

func printSuccess(msg string) {
	fmt.Println(successStyle.Render("✓ " + msg))
}

func printError(msg string) {
	fmt.Println(errorStyle.Render("✗ " + msg))
}

func printWarning(msg string) {
	fmt.Println(warningStyle.Render("⚠ " + msg))
}

func printInfo(msg string) {
	fmt.Println(dimStyle.Render("→ " + msg))
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}
