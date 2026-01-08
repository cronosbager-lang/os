package utils

import (
	"fmt"
	"os"
	"strings"
)

// ANSI color codes
const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"
	Blink     = "\033[5m"
	Reverse   = "\033[7m"
	Hidden    = "\033[8m"

	// Foreground colors
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"

	// Bright foreground colors
	BrightBlack   = "\033[90m"
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	BrightWhite   = "\033[97m"

	// Background colors
	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
)

var colorsEnabled = true

func init() {
	// Disable colors if NO_COLOR is set or not a terminal
	if os.Getenv("NO_COLOR") != "" {
		colorsEnabled = false
	}
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		colorsEnabled = false
	}
}

// EnableColors enables color output
func EnableColors() {
	colorsEnabled = true
}

// DisableColors disables color output
func DisableColors() {
	colorsEnabled = false
}

// ColorsEnabled returns whether colors are enabled
func ColorsEnabled() bool {
	return colorsEnabled
}

// Colorize applies color to text
func Colorize(text string, color string) string {
	if !colorsEnabled {
		return text
	}
	return color + text + Reset
}

// Color helper functions
func ColorRed(text string) string     { return Colorize(text, Red) }
func ColorGreen(text string) string   { return Colorize(text, Green) }
func ColorYellow(text string) string  { return Colorize(text, Yellow) }
func ColorBlue(text string) string    { return Colorize(text, Blue) }
func ColorMagenta(text string) string { return Colorize(text, Magenta) }
func ColorCyan(text string) string    { return Colorize(text, Cyan) }
func ColorWhite(text string) string   { return Colorize(text, White) }
func ColorBold(text string) string    { return Colorize(text, Bold) }
func ColorDim(text string) string     { return Colorize(text, Dim) }

// Icons
const (
	IconSuccess = "✓"
	IconError   = "✗"
	IconWarning = "⚠"
	IconInfo    = "→"
	IconDot     = "•"
	IconArrow   = "➜"
	IconCheck   = "✔"
	IconCross   = "✘"
	IconStar    = "★"
	IconHeart   = "♥"
)

// PrintSuccess prints a success message
func PrintSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", ColorGreen(IconSuccess), msg)
}

// PrintError prints an error message
func PrintError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s %s\n", ColorRed(IconError), msg)
}

// PrintWarning prints a warning message
func PrintWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", ColorYellow(IconWarning), msg)
}

// PrintInfo prints an info message
func PrintInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", ColorCyan(IconInfo), msg)
}

// PrintHeader prints a header
func PrintHeader(text string) {
	width := 50
	padding := (width - len(text) - 2) / 2
	line := strings.Repeat("═", width)

	fmt.Println(ColorCyan(line))
	fmt.Printf("%s %s %s\n",
		ColorCyan(strings.Repeat(" ", padding)),
		ColorBold(text),
		ColorCyan(strings.Repeat(" ", padding)))
	fmt.Println(ColorCyan(line))
}

// PrintDivider prints a divider line
func PrintDivider() {
	fmt.Println(ColorDim(strings.Repeat("─", 50)))
}

// PrintKeyValue prints a key-value pair
func PrintKeyValue(key, value string) {
	fmt.Printf("%s: %s\n", ColorBold(key), value)
}

// PrintList prints a list item
func PrintList(item string) {
	fmt.Printf("  %s %s\n", ColorCyan(IconDot), item)
}

// PrintTable prints a simple table
func PrintTable(headers []string, rows [][]string) {
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, h := range headers {
		fmt.Printf("%s%s  ", ColorBold(h), strings.Repeat(" ", widths[i]-len(h)))
	}
	fmt.Println()

	// Print separator
	for i := range headers {
		fmt.Printf("%s  ", strings.Repeat("─", widths[i]))
	}
	fmt.Println()

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				fmt.Printf("%s%s  ", cell, strings.Repeat(" ", widths[i]-len(cell)))
			}
		}
		fmt.Println()
	}
}

// StatusColor returns the appropriate color for a status
func StatusColor(status string) string {
	switch strings.ToLower(status) {
	case "running", "active", "success", "ok", "healthy":
		return Green
	case "stopped", "inactive", "failed", "error", "unhealthy":
		return Red
	case "pending", "starting", "stopping", "warning":
		return Yellow
	default:
		return White
	}
}

// ColorStatus colors a status string appropriately
func ColorStatus(status string) string {
	return Colorize(status, StatusColor(status))
}
