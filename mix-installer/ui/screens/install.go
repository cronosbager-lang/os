package screens

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mixos/mix-installer/pkg/models"
	"github.com/mixos/mix-installer/ui/components"
)

// InstallScreen is the installation progress screen
type InstallScreen struct {
	width       int
	height      int
	progress    float64
	phase       string
	step        string
	steps       *components.StepProgress
	spinner     *components.Spinner
	startTime   time.Time
	complete    bool
	success     bool
	result      *models.InstallResult
	progressCh  <-chan models.InstallProgress
}

// ProgressMsg is sent when progress updates
type ProgressMsg models.InstallProgress

// TickMsg is sent for spinner animation
type TickMsg time.Time

// NewInstallScreen creates a new installation screen
func NewInstallScreen(progressCh <-chan models.InstallProgress) *InstallScreen {
	stepNames := []string{
		"Preparing disk",
		"Creating partitions",
		"Formatting partitions",
		"Mounting filesystems",
		"Installing base system",
		"Installing packages",
		"Configuring system",
		"Creating user",
		"Installing bootloader",
		"Finalizing",
	}

	return &InstallScreen{
		width:      80,
		height:     24,
		steps:      components.NewStepProgress(stepNames),
		spinner:    components.NewSpinner(),
		startTime:  time.Now(),
		progressCh: progressCh,
	}
}

// Init initializes the screen
func (i *InstallScreen) Init() tea.Cmd {
	return tea.Batch(
		i.waitForProgress(),
		i.tickSpinner(),
	)
}

func (i *InstallScreen) waitForProgress() tea.Cmd {
	return func() tea.Msg {
		if i.progressCh == nil {
			return nil
		}
		progress, ok := <-i.progressCh
		if !ok {
			return nil
		}
		return ProgressMsg(progress)
	}
}

func (i *InstallScreen) tickSpinner() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Update handles messages
func (i *InstallScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		i.width = msg.Width
		i.height = msg.Height

	case ProgressMsg:
		i.updateProgress(models.InstallProgress(msg))
		return i, i.waitForProgress()

	case TickMsg:
		i.spinner.Tick()
		return i, i.tickSpinner()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// Allow cancel but warn
			return i, tea.Quit
		}
	}

	return i, nil
}

func (i *InstallScreen) updateProgress(progress models.InstallProgress) {
	i.phase = progress.Phase
	i.step = progress.Step
	i.progress = progress.Progress

	// Update step progress
	stepIndex := i.getStepIndex(progress.Phase)
	if stepIndex >= 0 {
		if progress.Error != "" {
			i.steps.SetStepStatus(stepIndex, components.StepFailed, progress.Error)
		} else if progress.Step == "Complete" {
			i.steps.SetStepStatus(stepIndex, components.StepComplete, "")
		} else {
			i.steps.SetStepStatus(stepIndex, components.StepRunning, progress.Step)
		}
	}

	// Check for completion
	if progress.Progress >= 100 || progress.CompletedAt > 0 {
		i.complete = true
		i.success = progress.Error == ""
	}
}

func (i *InstallScreen) getStepIndex(phase string) int {
	phases := map[string]int{
		"Preparing disk":         0,
		"Creating partitions":    1,
		"Formatting partitions":  2,
		"Mounting filesystems":   3,
		"Installing base system": 4,
		"Installing packages":    5,
		"Configuring system":     6,
		"Creating user":          7,
		"Installing bootloader":  8,
		"Finalizing":             9,
	}
	if idx, ok := phases[phase]; ok {
		return idx
	}
	return -1
}

// View renders the screen
func (i *InstallScreen) View() string {
	var b strings.Builder

	// Header
	b.WriteString(components.SmallHeader("Installing MIXOS GO", i.width))
	b.WriteString("\n\n")

	if i.complete {
		b.WriteString(i.renderComplete())
	} else {
		b.WriteString(i.renderProgress())
	}

	// Footer
	b.WriteString("\n")
	if i.complete {
		b.WriteString(components.Footer(components.CompleteFooter(), i.width))
	} else {
		b.WriteString(components.Footer(components.InstallFooter(), i.width))
	}

	return b.String()
}

func (i *InstallScreen) renderProgress() string {
	var b strings.Builder

	// Current phase with spinner
	phaseStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Bold(true)

	b.WriteString(i.spinner.Render())
	b.WriteString(" ")
	b.WriteString(phaseStyle.Render(i.phase))
	if i.step != "" && i.step != "Complete" {
		b.WriteString(": ")
		b.WriteString(i.step)
	}
	b.WriteString("\n\n")

	// Progress bar
	progressBar := components.NewProgressBar(i.width - 10)
	progressBar.SetProgress(i.progress)
	b.WriteString(lipgloss.NewStyle().
		Width(i.width).
		Align(lipgloss.Center).
		Render(progressBar.Render()))
	b.WriteString("\n\n")

	// Step progress
	stepBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		Padding(1, 2).
		Width(i.width - 20)

	b.WriteString(lipgloss.NewStyle().
		Width(i.width).
		Align(lipgloss.Center).
		Render(stepBox.Render(i.steps.Render())))

	// Elapsed time
	b.WriteString("\n\n")
	elapsed := time.Since(i.startTime)
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	b.WriteString(lipgloss.NewStyle().
		Width(i.width).
		Align(lipgloss.Center).
		Render(timeStyle.Render(fmt.Sprintf("Elapsed: %s", formatDuration(elapsed)))))

	return b.String()
}

func (i *InstallScreen) renderComplete() string {
	var b strings.Builder

	if i.success {
		// Success message
		successStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true)

		successBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#10B981")).
			Padding(2, 4).
			Width(i.width - 20)

		content := successStyle.Render("✓ Installation Complete!") + "\n\n"
		content += "MIXOS GO has been successfully installed on your system.\n\n"
		content += fmt.Sprintf("Duration: %s\n", formatDuration(time.Since(i.startTime)))
		content += "\nYou can now reboot into your new system."

		b.WriteString(lipgloss.NewStyle().
			Width(i.width).
			Align(lipgloss.Center).
			Render(successBox.Render(content)))

		// Next steps
		b.WriteString("\n\n")
		b.WriteString(components.SectionTitle("Next Steps"))
		b.WriteString("\n")
		b.WriteString("  1. Remove the installation media\n")
		b.WriteString("  2. Press 'r' to reboot\n")
		b.WriteString("  3. Log in with your created user account\n")
	} else {
		// Failure message
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true)

		errorBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#EF4444")).
			Padding(2, 4).
			Width(i.width - 20)

		content := errorStyle.Render("✗ Installation Failed") + "\n\n"
		content += "An error occurred during installation.\n"

		if i.result != nil && len(i.result.Errors) > 0 {
			content += "\nErrors:\n"
			for _, err := range i.result.Errors {
				content += "  • " + err + "\n"
			}
		}

		b.WriteString(lipgloss.NewStyle().
			Width(i.width).
			Align(lipgloss.Center).
			Render(errorBox.Render(content)))

		// Recovery options
		b.WriteString("\n\n")
		b.WriteString(components.SectionTitle("Recovery Options"))
		b.WriteString("\n")
		b.WriteString("  • Press 's' to drop to a shell for manual recovery\n")
		b.WriteString("  • Press 'q' to quit and try again\n")
	}

	return b.String()
}

// SetResult sets the installation result
func (i *InstallScreen) SetResult(result *models.InstallResult) {
	i.result = result
	i.complete = true
	i.success = result.Success
}

// IsComplete returns true if installation is complete
func (i *InstallScreen) IsComplete() bool {
	return i.complete
}

// IsSuccess returns true if installation was successful
func (i *InstallScreen) IsSuccess() bool {
	return i.success
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
