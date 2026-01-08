package screens

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mixos/mix-installer/pkg/models"
	"github.com/mixos/mix-installer/ui/components"
)

// SummaryScreen shows installation summary before proceeding
type SummaryScreen struct {
	width     int
	height    int
	plan      *models.InstallPlan
	confirmed bool
	cancelled bool
}

// NewSummaryScreen creates a new summary screen
func NewSummaryScreen(plan *models.InstallPlan) *SummaryScreen {
	return &SummaryScreen{
		width:  80,
		height: 24,
		plan:   plan,
	}
}

// Init initializes the screen
func (s *SummaryScreen) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (s *SummaryScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			s.confirmed = true
			return s, nil
		case "n", "N", "esc":
			s.cancelled = true
			return s, nil
		case "q", "ctrl+c":
			return s, tea.Quit
		}
	}

	return s, nil
}

// View renders the screen
func (s *SummaryScreen) View() string {
	var b strings.Builder

	// Header
	b.WriteString(components.SmallHeader("Installation Summary", s.width))
	b.WriteString("\n\n")

	// Step indicator
	steps := []string{"Mode", "Disk", "Partitions", "User", "Install"}
	b.WriteString(components.StepIndicator(4, 5, steps))
	b.WriteString("\n\n")

	// Summary box
	summaryBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		Padding(1, 2).
		Width(s.width - 10)

	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(summaryBox.Render(s.renderSummary())))

	// Warnings
	if len(s.plan.Warnings) > 0 {
		b.WriteString("\n\n")
		for _, warning := range s.plan.Warnings {
			b.WriteString(components.WarningBox(warning, s.width-20))
			b.WriteString("\n")
		}
	}

	// Confirmation
	b.WriteString("\n")
	confirmStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B")).
		Bold(true)

	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render(confirmStyle.Render("⚠ This will ERASE all data on the selected disk!")))

	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().
		Width(s.width).
		Align(lipgloss.Center).
		Render("Proceed with installation? [y/N]"))

	// Footer
	b.WriteString("\n\n")
	b.WriteString(components.Footer(components.ConfirmFooter(), s.width))

	return b.String()
}

func (s *SummaryScreen) renderSummary() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))

	// Disk
	b.WriteString(titleStyle.Render("📀 Disk"))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Device:     "))
	b.WriteString(valueStyle.Render(s.plan.Config.Disk.Device))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Model:      "))
	b.WriteString(valueStyle.Render(s.plan.Config.Disk.Model))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Size:       "))
	b.WriteString(valueStyle.Render(formatSize(s.plan.Config.Disk.Size)))
	b.WriteString("\n\n")

	// Partitions
	b.WriteString(titleStyle.Render("📁 Partitions"))
	b.WriteString("\n")
	for _, part := range s.plan.Config.Partitions {
		b.WriteString(fmt.Sprintf("  %s → %s (%s, %s)\n",
			part.Device, part.MountPoint, part.FSType, part.Size))
	}
	b.WriteString("\n")

	// System
	b.WriteString(titleStyle.Render("⚙️ System"))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Hostname:   "))
	b.WriteString(valueStyle.Render(s.plan.Config.System.Hostname))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Timezone:   "))
	b.WriteString(valueStyle.Render(s.plan.Config.System.Timezone))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Locale:     "))
	b.WriteString(valueStyle.Render(s.plan.Config.System.Locale))
	b.WriteString("\n\n")

	// User
	if s.plan.Config.User.Username != "" {
		b.WriteString(titleStyle.Render("👤 User"))
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("  Username:   "))
		b.WriteString(valueStyle.Render(s.plan.Config.User.Username))
		b.WriteString("\n")
		if s.plan.Config.User.FullName != "" {
			b.WriteString(labelStyle.Render("  Full Name:  "))
			b.WriteString(valueStyle.Render(s.plan.Config.User.FullName))
			b.WriteString("\n")
		}
		b.WriteString(labelStyle.Render("  Groups:     "))
		b.WriteString(valueStyle.Render(strings.Join(s.plan.Config.User.Groups, ", ")))
		b.WriteString("\n\n")
	}

	// Packages
	b.WriteString(titleStyle.Render("📦 Packages"))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Profile:    "))
	b.WriteString(valueStyle.Render(s.plan.Config.Packages.Profile))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Base:       "))
	b.WriteString(valueStyle.Render(fmt.Sprintf("%d packages", len(s.plan.Config.Packages.Base))))
	b.WriteString("\n")
	if len(s.plan.Config.Packages.Additional) > 0 {
		b.WriteString(labelStyle.Render("  Additional: "))
		b.WriteString(valueStyle.Render(fmt.Sprintf("%d packages", len(s.plan.Config.Packages.Additional))))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Bootloader
	b.WriteString(titleStyle.Render("🚀 Bootloader"))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Type:       "))
	b.WriteString(valueStyle.Render(s.plan.Config.Bootloader.Type))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Target:     "))
	b.WriteString(valueStyle.Render(s.plan.Config.Bootloader.Target))
	b.WriteString("\n\n")

	// Estimates
	b.WriteString(titleStyle.Render("⏱️ Estimates"))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Time:       "))
	b.WriteString(valueStyle.Render(fmt.Sprintf("~%d minutes", s.plan.EstimatedTime)))
	b.WriteString("\n")
	b.WriteString(labelStyle.Render("  Risk:       "))

	riskColor := lipgloss.Color("#10B981")
	if s.plan.RiskLevel == "medium" {
		riskColor = lipgloss.Color("#F59E0B")
	} else if s.plan.RiskLevel == "high" {
		riskColor = lipgloss.Color("#EF4444")
	}
	b.WriteString(lipgloss.NewStyle().Foreground(riskColor).Render(s.plan.RiskLevel))

	return b.String()
}

// IsConfirmed returns true if user confirmed
func (s *SummaryScreen) IsConfirmed() bool {
	return s.confirmed
}

// IsCancelled returns true if user cancelled
func (s *SummaryScreen) IsCancelled() bool {
	return s.cancelled
}
