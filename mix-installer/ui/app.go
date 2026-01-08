package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Screen int

const (
	ScreenWelcome Screen = iota
	ScreenMode
	ScreenDisk
	ScreenPartition
	ScreenUser
	ScreenPackages
	ScreenInstall
	ScreenComplete
)

type InstallConfig struct {
	Mode       string // "autonomous", "guided", "manual"
	Disk       string
	Partitions []Partition
	Username   string
	Password   string
	Hostname   string
	Timezone   string
	Packages   []string
}

type Partition struct {
	Device     string
	MountPoint string
	Size       string
	FSType     string
}

type App struct {
	screen    Screen
	config    InstallConfig
	spinner   spinner.Model
	width     int
	height    int
	err       error
	progress  float64
	statusMsg string
}

func NewApp() *App {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &App{
		screen:  ScreenWelcome,
		spinner: s,
		config: InstallConfig{
			Mode:     "guided",
			Packages: []string{"base", "kernel", "systemd", "docker", "mix-cli", "mix-agent"},
		},
	}
}

func (a *App) Init() tea.Cmd {
	return a.spinner.Tick
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return a, tea.Quit
		case "enter":
			return a.handleEnter()
		case "up", "k":
			return a.handleUp()
		case "down", "j":
			return a.handleDown()
		case "1", "2", "3":
			return a.handleNumber(msg.String())
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		return a, cmd

	case installProgressMsg:
		a.progress = msg.progress
		a.statusMsg = msg.status
		if msg.done {
			a.screen = ScreenComplete
		}
		return a, nil
	}

	return a, nil
}

func (a *App) View() string {
	switch a.screen {
	case ScreenWelcome:
		return a.viewWelcome()
	case ScreenMode:
		return a.viewMode()
	case ScreenDisk:
		return a.viewDisk()
	case ScreenUser:
		return a.viewUser()
	case ScreenInstall:
		return a.viewInstall()
	case ScreenComplete:
		return a.viewComplete()
	default:
		return "Unknown screen"
	}
}

func (a *App) handleEnter() (*App, tea.Cmd) {
	switch a.screen {
	case ScreenWelcome:
		a.screen = ScreenMode
	case ScreenMode:
		a.screen = ScreenDisk
	case ScreenDisk:
		a.screen = ScreenUser
	case ScreenUser:
		a.screen = ScreenInstall
		return a, a.startInstall()
	case ScreenComplete:
		return a, tea.Quit
	}
	return a, nil
}

func (a *App) handleUp() (*App, tea.Cmd) {
	return a, nil
}

func (a *App) handleDown() (*App, tea.Cmd) {
	return a, nil
}

func (a *App) handleNumber(n string) (*App, tea.Cmd) {
	if a.screen == ScreenMode {
		switch n {
		case "1":
			a.config.Mode = "autonomous"
		case "2":
			a.config.Mode = "guided"
		case "3":
			a.config.Mode = "manual"
		}
		a.screen = ScreenDisk
	}
	return a, nil
}

type installProgressMsg struct {
	progress float64
	status   string
	done     bool
}

func (a *App) startInstall() tea.Cmd {
	return func() tea.Msg {
		// Simulate installation
		return installProgressMsg{progress: 1.0, status: "Complete", done: true}
	}
}
