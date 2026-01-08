package screens

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mixos/mix-installer/pkg/models"
	"github.com/mixos/mix-installer/ui/components"
)

// DiskScreen is the disk selection screen
type DiskScreen struct {
	width        int
	height       int
	disks        []models.DiskInfo
	cursor       int
	selected     *models.DiskInfo
	recommendation *models.AIRecommendation
	showDetails  bool
}

// NewDiskScreen creates a new disk selection screen
func NewDiskScreen(disks []models.DiskInfo, recommendation *models.AIRecommendation) *DiskScreen {
	return &DiskScreen{
		width:          80,
		height:         24,
		disks:          disks,
		cursor:         0,
		recommendation: recommendation,
	}
}

// Init initializes the screen
func (d *DiskScreen) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (d *DiskScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if d.cursor > 0 {
				d.cursor--
			}
		case "down", "j":
			if d.cursor < len(d.disks)-1 {
				d.cursor++
			}
		case "enter":
			if d.cursor < len(d.disks) {
				d.selected = &d.disks[d.cursor]
			}
			return d, nil
		case "d":
			d.showDetails = !d.showDetails
		case "esc":
			return d, nil
		case "q", "ctrl+c":
			return d, tea.Quit
		}
	}

	return d, nil
}

// View renders the screen
func (d *DiskScreen) View() string {
	var b strings.Builder

	// Header
	b.WriteString(components.SmallHeader("Disk Selection", d.width))
	b.WriteString("\n\n")

	// Step indicator
	steps := []string{"Mode", "Disk", "Partitions", "User", "Install"}
	b.WriteString(components.StepIndicator(1, 5, steps))
	b.WriteString("\n\n")

	// AI Recommendation
	if d.recommendation != nil {
		b.WriteString(d.renderRecommendation())
		b.WriteString("\n\n")
	}

	// Section title
	b.WriteString(components.SectionTitle("Available Disks"))
	b.WriteString("\n")

	// Disk list
	for i, disk := range d.disks {
		b.WriteString(d.renderDisk(i, disk))
		b.WriteString("\n")
	}

	// Details panel
	if d.showDetails && d.cursor < len(d.disks) {
		b.WriteString("\n")
		b.WriteString(d.renderDetails(d.disks[d.cursor]))
	}

	// Footer
	b.WriteString("\n")
	bindings := []components.KeyBinding{
		{Key: "↑/↓", Description: "Navigate"},
		{Key: "Enter", Description: "Select"},
		{Key: "d", Description: "Details"},
		{Key: "Esc", Description: "Back"},
	}
	b.WriteString(components.Footer(bindings, d.width))

	return b.String()
}

func (d *DiskScreen) renderRecommendation() string {
	if d.recommendation == nil {
		return ""
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#10B981")).
		Padding(1, 2).
		Width(d.width - 10)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981")).
		Bold(true)

	content := titleStyle.Render("🤖 AI Recommendation") + "\n\n"

	if disk, ok := d.recommendation.Value.(models.DiskConfig); ok {
		content += fmt.Sprintf("Recommended: %s (%s)\n", disk.Device, disk.Model)
	}
	content += fmt.Sprintf("Confidence: %.0f%%\n", d.recommendation.Confidence*100)
	content += fmt.Sprintf("Reason: %s", d.recommendation.Reason)

	return lipgloss.NewStyle().
		Width(d.width).
		Align(lipgloss.Center).
		Render(style.Render(content))
}

func (d *DiskScreen) renderDisk(index int, disk models.DiskInfo) string {
	var style lipgloss.Style
	var prefix string

	isRecommended := false
	if d.recommendation != nil {
		if recDisk, ok := d.recommendation.Value.(models.DiskConfig); ok {
			isRecommended = recDisk.Device == disk.Device
		}
	}

	if index == d.cursor {
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7C3AED")).
			Bold(true).
			Padding(0, 1).
			Width(d.width - 10)
		prefix = "▸ "
	} else {
		style = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1).
			Width(d.width - 10)
		prefix = "  "
	}

	icon := "💾"
	if disk.Type == "nvme" {
		icon = "⚡"
	} else if disk.Removable {
		icon = "📱"
	}

	badge := ""
	if isRecommended {
		badge = " " + components.SuccessBadge("Recommended")
	}

	line := fmt.Sprintf("%s%s %s - %s (%s)%s",
		prefix, icon, disk.Device, disk.Model, formatSize(disk.Size), badge)

	return lipgloss.NewStyle().
		Width(d.width).
		Align(lipgloss.Center).
		Render(style.Render(line))
}

func (d *DiskScreen) renderDetails(disk models.DiskInfo) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6B7280")).
		Padding(1, 2).
		Width(d.width - 10)

	titleStyle := lipgloss.NewStyle().Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))

	var content strings.Builder
	content.WriteString(titleStyle.Render("Disk Details"))
	content.WriteString("\n\n")

	content.WriteString(labelStyle.Render("Device:     "))
	content.WriteString(disk.Device)
	content.WriteString("\n")

	content.WriteString(labelStyle.Render("Model:      "))
	content.WriteString(disk.Model)
	content.WriteString("\n")

	content.WriteString(labelStyle.Render("Size:       "))
	content.WriteString(formatSize(disk.Size))
	content.WriteString("\n")

	content.WriteString(labelStyle.Render("Type:       "))
	content.WriteString(disk.Type)
	content.WriteString("\n")

	content.WriteString(labelStyle.Render("Transport:  "))
	content.WriteString(disk.Transport)
	content.WriteString("\n")

	if len(disk.Partitions) > 0 {
		content.WriteString("\n")
		content.WriteString(titleStyle.Render("Existing Partitions"))
		content.WriteString("\n")

		for _, part := range disk.Partitions {
			content.WriteString(fmt.Sprintf("  %s: %s %s\n",
				part.Device, formatSize(part.Size), part.FSType))
		}
	} else {
		content.WriteString("\n")
		content.WriteString(labelStyle.Render("No existing partitions"))
	}

	return lipgloss.NewStyle().
		Width(d.width).
		Align(lipgloss.Center).
		Render(style.Render(content.String()))
}

// GetSelectedDisk returns the selected disk
func (d *DiskScreen) GetSelectedDisk() *models.DiskInfo {
	return d.selected
}

// IsSelected returns true if a disk was selected
func (d *DiskScreen) IsSelected() bool {
	return d.selected != nil
}

func formatSize(bytes uint64) string {
	const (
		GB = 1024 * 1024 * 1024
		TB = GB * 1024
	)

	if bytes >= TB {
		return fmt.Sprintf("%.1f TB", float64(bytes)/float64(TB))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
}
