package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const logo = `
 __  __ _____  _____  ____   _____    _____  ____  
|  \/  |_   _||  _  |/ __ \ / ____|  / ____|/ __ \ 
| \  / | | |  | |_| | |  | | (___   | |  __| |  | |
| |\/| | | |  |  _  | |  | |\___ \  | | |_ | |  | |
| |  | |_| |_ | | | | |__| |____) | | |__| | |__| |
|_|  |_|_____||_| |_|\____/|_____/   \_____|\____/ 
`

func (a *App) viewWelcome() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(logo))
	b.WriteString("\n\n")
	b.WriteString(titleStyle.Render("Welcome to MIXOS GO Installer"))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render("AI-Powered Operating System for Developers"))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("This installer will guide you through the installation process."))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("The AI agent can help automate the installation if desired."))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Press ENTER to continue • Press Q to quit"))

	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		b.String(),
	)
}

func (a *App) viewMode() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Installation Mode"))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render("Choose how you want to install MIXOS GO:"))
	b.WriteString("\n\n")

	modes := []struct {
		key  string
		name string
		desc string
	}{
		{"1", "Autonomous", "AI handles everything automatically"},
		{"2", "Guided", "AI assists with your input"},
		{"3", "Manual", "Full control over installation"},
	}

	for _, mode := range modes {
		style := normalStyle
		if (mode.key == "1" && a.config.Mode == "autonomous") ||
			(mode.key == "2" && a.config.Mode == "guided") ||
			(mode.key == "3" && a.config.Mode == "manual") {
			style = selectedStyle
		}

		b.WriteString(fmt.Sprintf("  %s  %s\n", style.Render("["+mode.key+"]"), style.Render(mode.name)))
		b.WriteString(fmt.Sprintf("      %s\n\n", dimStyle.Render(mode.desc)))
	}

	b.WriteString(helpStyle.Render("Press 1, 2, or 3 to select • Press ENTER to continue"))

	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		b.String(),
	)
}

func (a *App) viewDisk() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Disk Selection"))
	b.WriteString("\n\n")
	b.WriteString(normalStyle.Render("Select the disk to install MIXOS GO:"))
	b.WriteString("\n\n")

	// Mock disk list
	disks := []struct {
		device string
		size   string
		model  string
	}{
		{"/dev/sda", "500 GB", "Samsung SSD 860"},
		{"/dev/nvme0n1", "1 TB", "Samsung 970 EVO Plus"},
	}

	for i, disk := range disks {
		style := normalStyle
		if i == 0 {
			style = selectedStyle
			a.config.Disk = disk.device
		}
		b.WriteString(fmt.Sprintf("  %s %s (%s)\n", style.Render("→"), style.Render(disk.device), disk.size))
		b.WriteString(fmt.Sprintf("    %s\n\n", dimStyle.Render(disk.model)))
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("⚠ WARNING: All data on the selected disk will be erased!"))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Press ↑/↓ to select • Press ENTER to continue"))

	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		b.String(),
	)
}

func (a *App) viewUser() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("User Configuration"))
	b.WriteString("\n\n")

	// Set defaults
	a.config.Username = "developer"
	a.config.Hostname = "mixos"

	b.WriteString(normalStyle.Render("Username:  ") + selectedStyle.Render(a.config.Username))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("Hostname:  ") + selectedStyle.Render(a.config.Hostname))
	b.WriteString("\n")
	b.WriteString(normalStyle.Render("Timezone:  ") + selectedStyle.Render("UTC"))
	b.WriteString("\n\n")

	b.WriteString(dimStyle.Render("(In full version, these would be editable)"))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Press ENTER to start installation"))

	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		b.String(),
	)
}

func (a *App) viewInstall() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Installing MIXOS GO"))
	b.WriteString("\n\n")

	// Progress bar
	width := 40
	filled := int(a.progress * float64(width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	b.WriteString(progressBarStyle.Render(bar))
	b.WriteString(fmt.Sprintf(" %.0f%%\n\n", a.progress*100))

	b.WriteString(a.spinner.View() + " " + a.statusMsg)
	b.WriteString("\n\n")

	steps := []string{
		"Partitioning disk",
		"Creating filesystems",
		"Installing base system",
		"Installing packages",
		"Configuring system",
		"Installing bootloader",
		"Setting up AI agent",
	}

	for i, step := range steps {
		progress := float64(i+1) / float64(len(steps))
		if a.progress >= progress {
			b.WriteString(successStyle.Render("  ✓ " + step))
		} else if a.progress >= progress-0.15 {
			b.WriteString(normalStyle.Render("  → " + step))
		} else {
			b.WriteString(dimStyle.Render("  ○ " + step))
		}
		b.WriteString("\n")
	}

	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		b.String(),
	)
}

func (a *App) viewComplete() string {
	var b strings.Builder

	b.WriteString(successStyle.Render("✓ Installation Complete!"))
	b.WriteString("\n\n")
	b.WriteString(titleStyle.Render("MIXOS GO has been installed successfully"))
	b.WriteString("\n\n")

	b.WriteString(normalStyle.Render("System Summary:"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  OS:       MIXOS GO 1.0"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Kernel:   Linux 6.6.10-mixos"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  User:     " + a.config.Username))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  Hostname: " + a.config.Hostname))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  AI Agent: Ready"))
	b.WriteString("\n\n")

	b.WriteString(normalStyle.Render("After reboot, the AI agent will be available:"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  mix agent chat     - Interactive chat"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  mix agent execute  - Run tasks"))
	b.WriteString("\n\n")

	b.WriteString(helpStyle.Render("Press ENTER to reboot • Press Q to exit"))

	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		b.String(),
	)
}
