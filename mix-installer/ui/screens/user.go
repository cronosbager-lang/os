package screens

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mixos/mix-installer/pkg/models"
	"github.com/mixos/mix-installer/ui/components"
)

// UserScreen is the user configuration screen
type UserScreen struct {
	width      int
	height     int
	form       *components.Form
	userConfig models.UserConfig
	submitted  bool
	errors     []string
}

// NewUserScreen creates a new user configuration screen
func NewUserScreen() *UserScreen {
	form := components.NewForm()

	username := components.NewTextInput("Username")
	username.Placeholder = "Enter username"
	username.Width = 30
	form.AddInput(username)

	fullName := components.NewTextInput("Full Name")
	fullName.Placeholder = "Enter full name (optional)"
	fullName.Width = 30
	form.AddInput(fullName)

	password := components.NewTextInput("Password")
	password.Placeholder = "Enter password"
	password.Password = true
	password.Width = 30
	form.AddInput(password)

	confirmPassword := components.NewTextInput("Confirm Password")
	confirmPassword.Placeholder = "Confirm password"
	confirmPassword.Password = true
	confirmPassword.Width = 30
	form.AddInput(confirmPassword)

	hostname := components.NewTextInput("Hostname")
	hostname.Placeholder = "mixos-workstation"
	hostname.Width = 30
	form.AddInput(hostname)

	return &UserScreen{
		width:  80,
		height: 24,
		form:   form,
	}
}

// Init initializes the screen
func (u *UserScreen) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (u *UserScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		u.width = msg.Width
		u.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			u.form.FocusNext()
		case "shift+tab":
			u.form.FocusPrev()
		case "enter":
			if u.validate() {
				u.submitted = true
				u.buildConfig()
			}
		case "esc":
			return u, nil
		case "ctrl+c":
			return u, tea.Quit
		default:
			// Handle text input
			if focused := u.form.GetFocused(); focused != nil {
				switch msg.String() {
				case "backspace":
					focused.DeleteChar()
				case "delete":
					focused.DeleteForward()
				case "left":
					focused.MoveLeft()
				case "right":
					focused.MoveRight()
				case "home":
					focused.MoveStart()
				case "end":
					focused.MoveEnd()
				default:
					if len(msg.String()) == 1 {
						focused.InsertChar(rune(msg.String()[0]))
					}
				}
			}
		}
	}

	return u, nil
}

// View renders the screen
func (u *UserScreen) View() string {
	var b strings.Builder

	// Header
	b.WriteString(components.SmallHeader("User Configuration", u.width))
	b.WriteString("\n\n")

	// Step indicator
	steps := []string{"Mode", "Disk", "Partitions", "User", "Install"}
	b.WriteString(components.StepIndicator(3, 5, steps))
	b.WriteString("\n\n")

	// Section title
	b.WriteString(components.SectionTitle("Create User Account"))
	b.WriteString("\n")
	b.WriteString(components.SectionSubtitle("Configure the primary user account for your system"))
	b.WriteString("\n\n")

	// Form
	formContent := u.form.Render()

	// Add password strength indicator
	if u.form.FocusIndex == 2 { // Password field
		password := u.form.Inputs[2].Value
		if password != "" {
			score, feedback := components.PasswordStrength(password)
			formContent += "\n" + components.RenderPasswordStrength(score)
			if feedback != "" {
				formContent += "\n" + components.HelpText(feedback)
			}
		}
	}

	formBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		Padding(1, 2).
		Width(u.width - 20)

	b.WriteString(lipgloss.NewStyle().
		Width(u.width).
		Align(lipgloss.Center).
		Render(formBox.Render(formContent)))

	// Errors
	if len(u.errors) > 0 {
		b.WriteString("\n\n")
		for _, err := range u.errors {
			b.WriteString(components.ErrorBox(err, u.width-20))
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(components.Footer(components.FormFooter(), u.width))

	return b.String()
}

func (u *UserScreen) validate() bool {
	u.errors = nil

	values := u.form.GetValues()

	// Username validation
	username := values["Username"]
	if username == "" {
		u.errors = append(u.errors, "Username is required")
	} else if len(username) < 3 {
		u.errors = append(u.errors, "Username must be at least 3 characters")
	} else if !isValidUsername(username) {
		u.errors = append(u.errors, "Username can only contain lowercase letters, numbers, and underscores")
	}

	// Password validation
	password := values["Password"]
	confirmPassword := values["Confirm Password"]

	if password == "" {
		u.errors = append(u.errors, "Password is required")
	} else if len(password) < 8 {
		u.errors = append(u.errors, "Password must be at least 8 characters")
	}

	if password != confirmPassword {
		u.errors = append(u.errors, "Passwords do not match")
	}

	// Hostname validation
	hostname := values["Hostname"]
	if hostname != "" && !isValidHostname(hostname) {
		u.errors = append(u.errors, "Invalid hostname format")
	}

	return len(u.errors) == 0
}

func (u *UserScreen) buildConfig() {
	values := u.form.GetValues()

	hostname := values["Hostname"]
	if hostname == "" {
		hostname = "mixos-workstation"
	}

	u.userConfig = models.UserConfig{
		Username: values["Username"],
		Password: values["Password"],
		FullName: values["Full Name"],
		Groups:   []string{"wheel", "docker", "audio", "video"},
		Shell:    "/bin/bash",
	}
}

// GetUserConfig returns the user configuration
func (u *UserScreen) GetUserConfig() models.UserConfig {
	return u.userConfig
}

// GetHostname returns the hostname
func (u *UserScreen) GetHostname() string {
	values := u.form.GetValues()
	hostname := values["Hostname"]
	if hostname == "" {
		return "mixos-workstation"
	}
	return hostname
}

// IsSubmitted returns true if the form was submitted
func (u *UserScreen) IsSubmitted() bool {
	return u.submitted
}

func isValidUsername(username string) bool {
	if len(username) == 0 {
		return false
	}
	for i, c := range username {
		if i == 0 {
			if c < 'a' || c > 'z' {
				return false
			}
		} else {
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
				return false
			}
		}
	}
	return true
}

func isValidHostname(hostname string) bool {
	if len(hostname) == 0 || len(hostname) > 63 {
		return false
	}
	for i, c := range hostname {
		if i == 0 || i == len(hostname)-1 {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				return false
			}
		} else {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
				return false
			}
		}
	}
	return true
}
