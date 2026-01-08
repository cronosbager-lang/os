package components

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED")
	secondaryColor = lipgloss.Color("#10B981")
	accentColor    = lipgloss.Color("#F59E0B")
	errorColor     = lipgloss.Color("#EF4444")
	mutedColor     = lipgloss.Color("#6B7280")
	bgColor        = lipgloss.Color("#1F2937")

	// Header styles
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Padding(0, 2).
			MarginBottom(1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginBottom(1)
)

// Header renders the installer header
func Header(width int) string {
	logo := `
 ███╗   ███╗██╗██╗  ██╗ ██████╗ ███████╗
 ████╗ ████║██║╚██╗██╔╝██╔═══██╗██╔════╝
 ██╔████╔██║██║ ╚███╔╝ ██║   ██║███████╗
 ██║╚██╔╝██║██║ ██╔██╗ ██║   ██║╚════██║
 ██║ ╚═╝ ██║██║██╔╝ ██╗╚██████╔╝███████║
 ╚═╝     ╚═╝╚═╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝`

	logoStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true)

	subtitle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render("AI-Powered Operating System Installer")

	header := lipgloss.JoinVertical(
		lipgloss.Center,
		logoStyle.Render(logo),
		"",
		subtitle,
	)

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(header)
}

// SmallHeader renders a compact header
func SmallHeader(title string, width int) string {
	return headerStyle.
		Width(width).
		Align(lipgloss.Center).
		Render("MIXOS GO - " + title)
}

// SectionTitle renders a section title
func SectionTitle(title string) string {
	return titleStyle.Render("▸ " + title)
}

// SectionSubtitle renders a section subtitle
func SectionSubtitle(text string) string {
	return subtitleStyle.Render(text)
}

// Divider renders a horizontal divider
func Divider(width int) string {
	line := ""
	for i := 0; i < width; i++ {
		line += "─"
	}
	return lipgloss.NewStyle().
		Foreground(mutedColor).
		Render(line)
}

// StepIndicator renders a step indicator
func StepIndicator(current, total int, labels []string) string {
	var steps string

	for i := 0; i < total; i++ {
		var style lipgloss.Style
		var indicator string

		if i < current {
			// Completed
			style = lipgloss.NewStyle().Foreground(secondaryColor)
			indicator = "✓"
		} else if i == current {
			// Current
			style = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
			indicator = "●"
		} else {
			// Future
			style = lipgloss.NewStyle().Foreground(mutedColor)
			indicator = "○"
		}

		label := ""
		if i < len(labels) {
			label = labels[i]
		}

		step := style.Render(indicator + " " + label)
		if i < total-1 {
			step += lipgloss.NewStyle().Foreground(mutedColor).Render(" → ")
		}
		steps += step
	}

	return steps
}
