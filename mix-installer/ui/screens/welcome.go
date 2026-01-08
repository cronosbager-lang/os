package screens

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mixos/mix-installer/ui/components"
)

// WelcomeScreen is the welcome/mode selection screen
type WelcomeScreen struct {
	width    int
	height   int
	cursor   int
	selected int
}

// InstallMode represents installation mode
type InstallMode int

const (
	ModeAutonomous InstallMode = iota
	ModeGuided
	ModeManual
)

var modeOptions = []struct {
	title       string
	description string
	icon        string
}{
	{
		title:       "Autonomous Installation",
		description: "AI analyzes your system and installs MIXOS automatically",
		icon:        "🤖",
	},
	{
		title:       "Guided Installation",
		description: "Step-by-step installation with AI recommendations",
		icon:        "📋",
	},
	{
		title:       "Manual Installation",
		description: "Full control over all installation options",
		icon:        "⚙️",
	},
}

// NewWelcomeScreen creates a new welcome screen
func NewWelcomeScreen() *WelcomeScreen {
	return &WelcomeScreen{
		width:    80,
		height:   24,
		cursor:   0,
		selected: -1,
	}
}

// Init initializes the screen
func (w *WelcomeScreen) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (w *WelcomeScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w.width = msg.Width
		w.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if w.cursor > 0 {
				w.cursor--
			}
		case "down", "j":
			if w.cursor < len(modeOptions)-1 {
				w.cursor++
			}
		case "enter":
			w.selected = w.cursor
			return w, tea.Quit
		case "q", "ctrl+c":
			return w, tea.Quit
		}
	}

	return w, nil
}

// View renders the screen
func (w *WelcomeScreen) View() string {
	var b strings.Builder

	// Header
	b.WriteString(components.Header(w.width))
	b.WriteString("\n\n")

	// Welcome message
	welcomeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Width(w.width).
		Align(lipgloss.Center)

	b.WriteString(welcomeStyle.Render("Welcome to MIXOS GO Installation"))
	b.WriteString("\n\n")

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF")).
		Width(w.width).
		Align(lipgloss.Center)

	b.WriteString(subtitleStyle.Render("Select an installation mode to begin"))
	b.WriteString("\n\n")

	// Mode options
	for i, opt := range modeOptions {
		b.WriteString(w.renderOption(i, opt.title, opt.description, opt.icon))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(components.Footer(components.DefaultFooter(), w.width))

	return b.String()
}

func (w *WelcomeScreen) renderOption(index int, title, description, icon string) string {
	var style lipgloss.Style
	var prefix string

	if index == w.cursor {
		style = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(1, 2).
			Width(w.width - 10)
		prefix = "▸ "
	} else {
		style = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#374151")).
			Padding(1, 2).
			Width(w.width - 10)
		prefix = "  "
	}

	titleStyle := lipgloss.NewStyle().Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))

	content := prefix + icon + " " + titleStyle.Render(title) + "\n" +
		"    " + descStyle.Render(description)

	return lipgloss.NewStyle().
		Width(w.width).
		Align(lipgloss.Center).
		Render(style.Render(content))
}

// GetSelectedMode returns the selected installation mode
func (w *WelcomeScreen) GetSelectedMode() InstallMode {
	return InstallMode(w.selected)
}

// IsSelected returns true if a mode was selected
func (w *WelcomeScreen) IsSelected() bool {
	return w.selected >= 0
}
