package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// KeyBinding represents a keyboard shortcut
type KeyBinding struct {
	Key         string
	Description string
}

// Footer renders the footer with key bindings
func Footer(bindings []KeyBinding, width int) string {
	var parts []string

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#374151")).
		Padding(0, 1)

	descStyle := lipgloss.NewStyle().
		Foreground(mutedColor)

	for _, binding := range bindings {
		part := keyStyle.Render(binding.Key) + " " + descStyle.Render(binding.Description)
		parts = append(parts, part)
	}

	content := strings.Join(parts, "  ")

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(content)
}

// DefaultFooter returns the default footer bindings
func DefaultFooter() []KeyBinding {
	return []KeyBinding{
		{Key: "↑/↓", Description: "Navigate"},
		{Key: "Enter", Description: "Select"},
		{Key: "q", Description: "Quit"},
	}
}

// NavigationFooter returns navigation footer bindings
func NavigationFooter() []KeyBinding {
	return []KeyBinding{
		{Key: "↑/↓", Description: "Navigate"},
		{Key: "Enter", Description: "Select"},
		{Key: "Esc", Description: "Back"},
		{Key: "q", Description: "Quit"},
	}
}

// FormFooter returns form footer bindings
func FormFooter() []KeyBinding {
	return []KeyBinding{
		{Key: "Tab", Description: "Next field"},
		{Key: "Shift+Tab", Description: "Previous"},
		{Key: "Enter", Description: "Submit"},
		{Key: "Esc", Description: "Cancel"},
	}
}

// ConfirmFooter returns confirmation footer bindings
func ConfirmFooter() []KeyBinding {
	return []KeyBinding{
		{Key: "y", Description: "Yes"},
		{Key: "n", Description: "No"},
		{Key: "Esc", Description: "Cancel"},
	}
}

// InstallFooter returns installation footer bindings
func InstallFooter() []KeyBinding {
	return []KeyBinding{
		{Key: "Ctrl+C", Description: "Cancel (not recommended)"},
	}
}

// CompleteFooter returns completion footer bindings
func CompleteFooter() []KeyBinding {
	return []KeyBinding{
		{Key: "r", Description: "Reboot"},
		{Key: "s", Description: "Shell"},
		{Key: "q", Description: "Quit"},
	}
}

// StatusBar renders a status bar
func StatusBar(left, center, right string, width int) string {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("#1F2937")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Width(width)

	leftStyle := lipgloss.NewStyle().Align(lipgloss.Left)
	centerStyle := lipgloss.NewStyle().Align(lipgloss.Center)
	rightStyle := lipgloss.NewStyle().Align(lipgloss.Right)

	// Calculate widths
	sideWidth := (width - len(center)) / 2
	if sideWidth < 0 {
		sideWidth = 0
	}

	content := leftStyle.Width(sideWidth).Render(left) +
		centerStyle.Render(center) +
		rightStyle.Width(sideWidth).Render(right)

	return style.Render(content)
}

// HelpText renders help text
func HelpText(text string) string {
	style := lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true)
	return style.Render(text)
}

// InfoBox renders an info box
func InfoBox(title, content string, width int) string {
	titleStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true)

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(1, 2).
		Width(width)

	inner := titleStyle.Render(title) + "\n\n" + contentStyle.Render(content)

	return boxStyle.Render(inner)
}

// WarningBox renders a warning box
func WarningBox(content string, width int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Foreground(accentColor).
		Padding(1, 2).
		Width(width)

	return style.Render("⚠ " + content)
}

// ErrorBox renders an error box
func ErrorBox(content string, width int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(errorColor).
		Foreground(errorColor).
		Padding(1, 2).
		Width(width)

	return style.Render("✗ " + content)
}

// SuccessBox renders a success box
func SuccessBox(content string, width int) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(secondaryColor).
		Foreground(secondaryColor).
		Padding(1, 2).
		Width(width)

	return style.Render("✓ " + content)
}

// Tooltip renders a tooltip
func Tooltip(text string) string {
	style := lipgloss.NewStyle().
		Foreground(mutedColor).
		Background(lipgloss.Color("#374151")).
		Padding(0, 1)

	return style.Render("💡 " + text)
}

// Badge renders a badge
func Badge(text string, color lipgloss.Color) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(color).
		Padding(0, 1).
		Bold(true)

	return style.Render(text)
}

// PrimaryBadge renders a primary badge
func PrimaryBadge(text string) string {
	return Badge(text, primaryColor)
}

// SuccessBadge renders a success badge
func SuccessBadge(text string) string {
	return Badge(text, secondaryColor)
}

// WarningBadge renders a warning badge
func WarningBadge(text string) string {
	return Badge(text, accentColor)
}

// ErrorBadge renders an error badge
func ErrorBadge(text string) string {
	return Badge(text, errorColor)
}
