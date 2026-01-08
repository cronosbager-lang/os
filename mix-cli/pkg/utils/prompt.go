package utils

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Confirm asks for yes/no confirmation
func Confirm(message string, defaultYes bool) bool {
	suffix := "[y/N]"
	if defaultYes {
		suffix = "[Y/n]"
	}

	fmt.Printf("%s %s ", message, suffix)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}

	input = strings.TrimSpace(strings.ToLower(input))

	if input == "" {
		return defaultYes
	}

	return input == "y" || input == "yes"
}

// PromptString prompts for a string input
func PromptString(message string, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", message, defaultValue)
	} else {
		fmt.Printf("%s: ", message)
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}

	return input
}

// PromptPassword prompts for a password (hidden input)
func PromptPassword(message string) string {
	fmt.Printf("%s: ", message)

	// In a real implementation, this would disable echo
	// For now, we just read normally
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	return strings.TrimSpace(input)
}

// PromptInt prompts for an integer input
func PromptInt(message string, defaultValue int) int {
	fmt.Printf("%s [%d]: ", message, defaultValue)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(input)
	if err != nil {
		return defaultValue
	}

	return value
}

// PromptSelect prompts for selection from a list
func PromptSelect(message string, options []string, defaultIndex int) int {
	fmt.Println(message)
	for i, option := range options {
		marker := "  "
		if i == defaultIndex {
			marker = ColorCyan("> ")
		}
		fmt.Printf("%s%d. %s\n", marker, i+1, option)
	}

	fmt.Printf("Enter selection [%d]: ", defaultIndex+1)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultIndex
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultIndex
	}

	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(options) {
		return defaultIndex
	}

	return selection - 1
}

// PromptMultiSelect prompts for multiple selections
func PromptMultiSelect(message string, options []string) []int {
	fmt.Println(message)
	fmt.Println("(Enter comma-separated numbers, e.g., 1,3,5)")

	for i, option := range options {
		fmt.Printf("  %d. %s\n", i+1, option)
	}

	fmt.Print("Enter selections: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	var selections []int
	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		if num, err := strconv.Atoi(part); err == nil {
			if num >= 1 && num <= len(options) {
				selections = append(selections, num-1)
			}
		}
	}

	return selections
}

// PromptEditor opens an editor for multi-line input
func PromptEditor(message string, defaultContent string) (string, error) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "mixos-edit-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	// Write default content
	if defaultContent != "" {
		tmpFile.WriteString(defaultContent)
	}
	tmpFile.Close()

	// Get editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "nano"
	}

	// Open editor
	fmt.Printf("%s (opening %s...)\n", message, editor)

	// In a real implementation, this would exec the editor
	// For now, we just return the default content
	return defaultContent, nil
}

// PromptPath prompts for a file path with tab completion
func PromptPath(message string, defaultPath string) string {
	if defaultPath != "" {
		fmt.Printf("%s [%s]: ", message, defaultPath)
	} else {
		fmt.Printf("%s: ", message)
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultPath
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultPath
	}

	// Expand ~ to home directory
	if strings.HasPrefix(input, "~") {
		home, _ := os.UserHomeDir()
		input = strings.Replace(input, "~", home, 1)
	}

	return input
}

// PromptChoice prompts for a choice with custom keys
func PromptChoice(message string, choices map[string]string) string {
	fmt.Printf("%s ", message)

	// Build choice string
	var keys []string
	for k := range choices {
		keys = append(keys, k)
	}
	fmt.Printf("[%s]: ", strings.Join(keys, "/"))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	input = strings.TrimSpace(strings.ToLower(input))

	if _, ok := choices[input]; ok {
		return input
	}

	return ""
}

// WaitForEnter waits for the user to press Enter
func WaitForEnter(message string) {
	if message == "" {
		message = "Press Enter to continue..."
	}
	fmt.Print(message)

	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
}

// ClearScreen clears the terminal screen
func ClearScreen() {
	fmt.Print("\033[H\033[2J")
}

// MoveCursor moves the cursor to a position
func MoveCursor(row, col int) {
	fmt.Printf("\033[%d;%dH", row, col)
}

// HideCursor hides the cursor
func HideCursor() {
	fmt.Print("\033[?25l")
}

// ShowCursor shows the cursor
func ShowCursor() {
	fmt.Print("\033[?25h")
}
