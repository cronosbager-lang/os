package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TextInput represents a text input field
type TextInput struct {
	Value       string
	Placeholder string
	Label       string
	Width       int
	Focused     bool
	Password    bool
	MaxLength   int
	CursorPos   int
	Error       string
}

// NewTextInput creates a new text input
func NewTextInput(label string) *TextInput {
	return &TextInput{
		Label:     label,
		Width:     40,
		MaxLength: 100,
	}
}

// Focus focuses the input
func (t *TextInput) Focus() {
	t.Focused = true
}

// Blur unfocuses the input
func (t *TextInput) Blur() {
	t.Focused = false
}

// SetValue sets the input value
func (t *TextInput) SetValue(value string) {
	if t.MaxLength > 0 && len(value) > t.MaxLength {
		value = value[:t.MaxLength]
	}
	t.Value = value
	t.CursorPos = len(value)
}

// InsertChar inserts a character at cursor position
func (t *TextInput) InsertChar(char rune) {
	if t.MaxLength > 0 && len(t.Value) >= t.MaxLength {
		return
	}
	t.Value = t.Value[:t.CursorPos] + string(char) + t.Value[t.CursorPos:]
	t.CursorPos++
}

// DeleteChar deletes character before cursor
func (t *TextInput) DeleteChar() {
	if t.CursorPos > 0 {
		t.Value = t.Value[:t.CursorPos-1] + t.Value[t.CursorPos:]
		t.CursorPos--
	}
}

// DeleteForward deletes character after cursor
func (t *TextInput) DeleteForward() {
	if t.CursorPos < len(t.Value) {
		t.Value = t.Value[:t.CursorPos] + t.Value[t.CursorPos+1:]
	}
}

// MoveLeft moves cursor left
func (t *TextInput) MoveLeft() {
	if t.CursorPos > 0 {
		t.CursorPos--
	}
}

// MoveRight moves cursor right
func (t *TextInput) MoveRight() {
	if t.CursorPos < len(t.Value) {
		t.CursorPos++
	}
}

// MoveStart moves cursor to start
func (t *TextInput) MoveStart() {
	t.CursorPos = 0
}

// MoveEnd moves cursor to end
func (t *TextInput) MoveEnd() {
	t.CursorPos = len(t.Value)
}

// Render renders the text input
func (t *TextInput) Render() string {
	var lines []string

	// Label
	if t.Label != "" {
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
		lines = append(lines, labelStyle.Render(t.Label))
	}

	// Input field
	var displayValue string
	if t.Password {
		displayValue = strings.Repeat("•", len(t.Value))
	} else {
		displayValue = t.Value
	}

	// Show placeholder if empty
	if displayValue == "" && t.Placeholder != "" && !t.Focused {
		displayValue = t.Placeholder
	}

	// Pad to width
	if len(displayValue) < t.Width {
		displayValue += strings.Repeat(" ", t.Width-len(displayValue))
	}

	// Add cursor if focused
	if t.Focused {
		if t.CursorPos < len(displayValue) {
			cursorStyle := lipgloss.NewStyle().
				Background(lipgloss.Color("#FFFFFF")).
				Foreground(lipgloss.Color("#000000"))
			displayValue = displayValue[:t.CursorPos] +
				cursorStyle.Render(string(displayValue[t.CursorPos])) +
				displayValue[t.CursorPos+1:]
		} else {
			displayValue += lipgloss.NewStyle().
				Background(lipgloss.Color("#FFFFFF")).
				Render(" ")
		}
	}

	// Style based on focus state
	var inputStyle lipgloss.Style
	if t.Focused {
		inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(0, 1)
	} else {
		inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(0, 1)
	}

	lines = append(lines, inputStyle.Render(displayValue))

	// Error message
	if t.Error != "" {
		errorStyle := lipgloss.NewStyle().Foreground(errorColor)
		lines = append(lines, errorStyle.Render("  "+t.Error))
	}

	return strings.Join(lines, "\n")
}

// Form represents a form with multiple inputs
type Form struct {
	Inputs      []*TextInput
	FocusIndex  int
	SubmitLabel string
}

// NewForm creates a new form
func NewForm() *Form {
	return &Form{
		SubmitLabel: "Submit",
	}
}

// AddInput adds an input to the form
func (f *Form) AddInput(input *TextInput) {
	f.Inputs = append(f.Inputs, input)
	if len(f.Inputs) == 1 {
		input.Focus()
	}
}

// FocusNext focuses the next input
func (f *Form) FocusNext() {
	if len(f.Inputs) == 0 {
		return
	}
	f.Inputs[f.FocusIndex].Blur()
	f.FocusIndex = (f.FocusIndex + 1) % len(f.Inputs)
	f.Inputs[f.FocusIndex].Focus()
}

// FocusPrev focuses the previous input
func (f *Form) FocusPrev() {
	if len(f.Inputs) == 0 {
		return
	}
	f.Inputs[f.FocusIndex].Blur()
	f.FocusIndex--
	if f.FocusIndex < 0 {
		f.FocusIndex = len(f.Inputs) - 1
	}
	f.Inputs[f.FocusIndex].Focus()
}

// GetFocused returns the focused input
func (f *Form) GetFocused() *TextInput {
	if f.FocusIndex >= 0 && f.FocusIndex < len(f.Inputs) {
		return f.Inputs[f.FocusIndex]
	}
	return nil
}

// GetValues returns all input values
func (f *Form) GetValues() map[string]string {
	values := make(map[string]string)
	for _, input := range f.Inputs {
		values[input.Label] = input.Value
	}
	return values
}

// Validate validates all inputs
func (f *Form) Validate(validators map[string]func(string) string) bool {
	valid := true
	for _, input := range f.Inputs {
		if validator, ok := validators[input.Label]; ok {
			if err := validator(input.Value); err != "" {
				input.Error = err
				valid = false
			} else {
				input.Error = ""
			}
		}
	}
	return valid
}

// Render renders the form
func (f *Form) Render() string {
	var lines []string

	for _, input := range f.Inputs {
		lines = append(lines, input.Render())
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// Confirm renders a confirmation dialog
func Confirm(message string, defaultYes bool) string {
	var lines []string

	msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	lines = append(lines, msgStyle.Render(message))
	lines = append(lines, "")

	var options string
	if defaultYes {
		options = "[Y/n]"
	} else {
		options = "[y/N]"
	}

	optStyle := lipgloss.NewStyle().Foreground(mutedColor)
	lines = append(lines, optStyle.Render(options))

	return strings.Join(lines, "\n")
}

// PasswordStrength calculates password strength
func PasswordStrength(password string) (int, string) {
	score := 0
	feedback := []string{}

	if len(password) >= 8 {
		score++
	} else {
		feedback = append(feedback, "at least 8 characters")
	}

	if len(password) >= 12 {
		score++
	}

	hasLower := false
	hasUpper := false
	hasDigit := false
	hasSpecial := false

	for _, c := range password {
		switch {
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}

	if hasLower {
		score++
	} else {
		feedback = append(feedback, "lowercase letter")
	}

	if hasUpper {
		score++
	} else {
		feedback = append(feedback, "uppercase letter")
	}

	if hasDigit {
		score++
	} else {
		feedback = append(feedback, "number")
	}

	if hasSpecial {
		score++
	}

	var message string
	if len(feedback) > 0 {
		message = "Add: " + strings.Join(feedback, ", ")
	}

	return score, message
}

// RenderPasswordStrength renders a password strength indicator
func RenderPasswordStrength(score int) string {
	var color lipgloss.Color
	var label string

	switch {
	case score <= 2:
		color = errorColor
		label = "Weak"
	case score <= 4:
		color = accentColor
		label = "Medium"
	default:
		color = secondaryColor
		label = "Strong"
	}

	bar := strings.Repeat("█", score) + strings.Repeat("░", 6-score)
	style := lipgloss.NewStyle().Foreground(color)

	return style.Render(bar + " " + label)
}
