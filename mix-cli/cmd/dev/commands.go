package dev

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup development environment",
	Long: `Setup a complete development environment using AI assistance.

This will analyze your system and install recommended tools.`,
	Run: runSetup,
}

var initCmd = &cobra.Command{
	Use:   "init [language]",
	Short: "Initialize a new project",
	Long: `Initialize a new project for the specified language.

Supported languages:
  python    Python project with virtualenv
  go        Go module
  rust      Rust/Cargo project
  node      Node.js project
  react     React application`,
	Args: cobra.ExactArgs(1),
	Run:  runInit,
}

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Install development tools",
	Long:  `Install common development tools.`,
	Run:   runTools,
}

func runSetup(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Println(cyan("═══════════════════════════════════════"))
	fmt.Println(cyan("    Development Environment Setup      "))
	fmt.Println(cyan("═══════════════════════════════════════"))
	fmt.Println()

	// Try to use AI agent for intelligent setup
	reqBody := map[string]interface{}{
		"task":         "setup development environment",
		"auto_confirm": true,
	}
	jsonBody, _ := json.Marshal(reqBody)

	resp, err := http.Post(
		"http://localhost:8765/api/execute",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)

	if err != nil {
		// Fallback to manual setup
		fmt.Printf("%s AI agent not available, using manual setup\n", cyan("→"))
		manualSetup()
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["status"] == "completed" {
		fmt.Printf("%s Development environment setup complete!\n", green("✓"))
	} else {
		manualSetup()
	}
}

func manualSetup() {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	tools := []struct {
		name    string
		check   string
		install string
	}{
		{"git", "git --version", "mix-pkg install git"},
		{"python3", "python3 --version", "mix-pkg install python"},
		{"go", "go version", "mix-pkg install go"},
		{"node", "node --version", "mix-pkg install nodejs"},
		{"docker", "docker --version", "mix-pkg install docker"},
	}

	for _, tool := range tools {
		fmt.Printf("%s Checking %s...\n", cyan("→"), tool.name)
		if err := exec.Command("sh", "-c", tool.check).Run(); err != nil {
			fmt.Printf("  Installing %s...\n", tool.name)
			exec.Command("sh", "-c", tool.install).Run()
		} else {
			fmt.Printf("  %s %s already installed\n", green("✓"), tool.name)
		}
	}
}

func runInit(cmd *cobra.Command, args []string) {
	language := strings.ToLower(args[0])
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	fmt.Printf("%s Initializing %s project...\n", cyan("→"), language)

	var initCmd *exec.Cmd

	switch language {
	case "python":
		initCmd = exec.Command("sh", "-c", `
			python3 -m venv venv
			echo "# Python Project" > README.md
			echo "venv/" > .gitignore
			echo "__pycache__/" >> .gitignore
			echo "*.pyc" >> .gitignore
			git init
		`)
	case "go":
		cwd, _ := os.Getwd()
		modName := "example.com/" + strings.ToLower(cwd[strings.LastIndex(cwd, "/")+1:])
		initCmd = exec.Command("sh", "-c", fmt.Sprintf(`
			go mod init %s
			echo "# Go Project" > README.md
			git init
		`, modName))
	case "rust":
		initCmd = exec.Command("cargo", "init")
	case "node", "nodejs":
		initCmd = exec.Command("npm", "init", "-y")
	case "react":
		initCmd = exec.Command("npx", "create-react-app", ".")
	default:
		fmt.Printf("%s Unknown language: %s\n", red("✗"), language)
		fmt.Println("Supported: python, go, rust, node, react")
		os.Exit(1)
	}

	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr

	if err := initCmd.Run(); err != nil {
		fmt.Printf("%s Failed to initialize project: %v\n", red("✗"), err)
		os.Exit(1)
	}

	fmt.Printf("%s Project initialized!\n", green("✓"))
}

func runTools(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Println(cyan("Installing development tools..."))

	tools := []string{
		"git",
		"vim",
		"tmux",
		"htop",
		"curl",
		"wget",
		"jq",
	}

	for _, tool := range tools {
		fmt.Printf("  %s Installing %s...\n", cyan("→"), tool)
		exec.Command("mix-pkg", "install", "-y", tool).Run()
	}

	fmt.Printf("%s Development tools installed!\n", green("✓"))
}
