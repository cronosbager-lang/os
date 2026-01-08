package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ProgressBar renders a progress bar
type ProgressBar struct {
	Progress float64 // 0-100
	Width    int
	Label    string
	ShowPct  bool
}

// NewProgressBar creates a new progress bar
func NewProgressBar(width int) *ProgressBar {
	return &ProgressBar{
		Progress: 0,
		Width:    width,
		ShowPct:  true,
	}
}

// SetProgress sets the progress value
func (p *ProgressBar) SetProgress(progress float64) {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	p.Progress = progress
}

// Render renders the progress bar
func (p *ProgressBar) Render() string {
	barWidth := p.Width - 10 // Leave room for percentage
	if p.Label != "" {
		barWidth -= len(p.Label) + 2
	}

	filled := int(float64(barWidth) * p.Progress / 100)
	empty := barWidth - filled

	filledStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(primaryColor)

	emptyStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Background(lipgloss.Color("#374151"))

	bar := filledStyle.Render(strings.Repeat("█", filled)) +
		emptyStyle.Render(strings.Repeat("░", empty))

	result := ""
	if p.Label != "" {
		result = p.Label + " "
	}
	result += "[" + bar + "]"

	if p.ShowPct {
		pctStyle := lipgloss.NewStyle().Foreground(secondaryColor).Bold(true)
		result += " " + pctStyle.Render(fmt.Sprintf("%3.0f%%", p.Progress))
	}

	return result
}

// Spinner renders a spinner animation
type Spinner struct {
	Frame int
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// NewSpinner creates a new spinner
func NewSpinner() *Spinner {
	return &Spinner{Frame: 0}
}

// Tick advances the spinner
func (s *Spinner) Tick() {
	s.Frame = (s.Frame + 1) % len(spinnerFrames)
}

// Render renders the spinner
func (s *Spinner) Render() string {
	style := lipgloss.NewStyle().Foreground(primaryColor)
	return style.Render(spinnerFrames[s.Frame])
}

// RenderWithLabel renders spinner with a label
func (s *Spinner) RenderWithLabel(label string) string {
	return s.Render() + " " + label
}

// StepProgress renders a step-by-step progress indicator
type StepProgress struct {
	Steps       []StepInfo
	CurrentStep int
	Width       int
}

// StepInfo represents a step
type StepInfo struct {
	Name    string
	Status  StepStatus
	Message string
}

// StepStatus represents step status
type StepStatus int

const (
	StepPending StepStatus = iota
	StepRunning
	StepComplete
	StepFailed
	StepSkipped
)

// NewStepProgress creates a new step progress
func NewStepProgress(steps []string) *StepProgress {
	stepInfos := make([]StepInfo, len(steps))
	for i, name := range steps {
		stepInfos[i] = StepInfo{Name: name, Status: StepPending}
	}
	return &StepProgress{
		Steps:       stepInfos,
		CurrentStep: 0,
		Width:       60,
	}
}

// SetStepStatus sets the status of a step
func (sp *StepProgress) SetStepStatus(index int, status StepStatus, message string) {
	if index >= 0 && index < len(sp.Steps) {
		sp.Steps[index].Status = status
		sp.Steps[index].Message = message
	}
}

// NextStep advances to the next step
func (sp *StepProgress) NextStep() {
	if sp.CurrentStep < len(sp.Steps) {
		sp.Steps[sp.CurrentStep].Status = StepComplete
		sp.CurrentStep++
		if sp.CurrentStep < len(sp.Steps) {
			sp.Steps[sp.CurrentStep].Status = StepRunning
		}
	}
}

// Render renders the step progress
func (sp *StepProgress) Render() string {
	var lines []string

	for i, step := range sp.Steps {
		line := sp.renderStep(i, step)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (sp *StepProgress) renderStep(index int, step StepInfo) string {
	var icon string
	var style lipgloss.Style

	switch step.Status {
	case StepPending:
		icon = "○"
		style = lipgloss.NewStyle().Foreground(mutedColor)
	case StepRunning:
		icon = "●"
		style = lipgloss.NewStyle().Foreground(primaryColor).Bold(true)
	case StepComplete:
		icon = "✓"
		style = lipgloss.NewStyle().Foreground(secondaryColor)
	case StepFailed:
		icon = "✗"
		style = lipgloss.NewStyle().Foreground(errorColor)
	case StepSkipped:
		icon = "○"
		style = lipgloss.NewStyle().Foreground(mutedColor).Strikethrough(true)
	}

	line := fmt.Sprintf("%s %s", icon, step.Name)
	result := style.Render(line)

	if step.Message != "" && (step.Status == StepRunning || step.Status == StepFailed) {
		msgStyle := lipgloss.NewStyle().Foreground(mutedColor)
		result += "\n" + msgStyle.Render("    "+step.Message)
	}

	return result
}

// InstallProgress renders installation progress
func InstallProgress(phase string, step string, progress float64, width int) string {
	var lines []string

	// Phase header
	phaseStyle := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true)
	lines = append(lines, phaseStyle.Render("Phase: "+phase))

	// Current step
	if step != "" {
		stepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
		lines = append(lines, stepStyle.Render("  → "+step))
	}

	// Progress bar
	lines = append(lines, "")
	bar := NewProgressBar(width - 4)
	bar.SetProgress(progress)
	lines = append(lines, "  "+bar.Render())

	return strings.Join(lines, "\n")
}

// CompletionSummary renders installation completion summary
func CompletionSummary(success bool, duration int64, warnings []string, errors []string) string {
	var lines []string

	if success {
		successStyle := lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)
		lines = append(lines, successStyle.Render("✓ Installation Complete!"))
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("Duration: %d minutes %d seconds",
			duration/60, duration%60))
	} else {
		errorStyle := lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)
		lines = append(lines, errorStyle.Render("✗ Installation Failed"))
	}

	if len(warnings) > 0 {
		lines = append(lines, "")
		warnStyle := lipgloss.NewStyle().Foreground(accentColor)
		lines = append(lines, warnStyle.Render("Warnings:"))
		for _, w := range warnings {
			lines = append(lines, "  ⚠ "+w)
		}
	}

	if len(errors) > 0 {
		lines = append(lines, "")
		errStyle := lipgloss.NewStyle().Foreground(errorColor)
		lines = append(lines, errStyle.Render("Errors:"))
		for _, e := range errors {
			lines = append(lines, "  ✗ "+e)
		}
	}

	return strings.Join(lines, "\n")
}
