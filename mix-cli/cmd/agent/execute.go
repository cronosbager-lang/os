package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var executeCmd = &cobra.Command{
	Use:   "execute [task]",
	Short: "Execute an autonomous task",
	Long: `Execute an autonomous task using the AI agent.

The agent will plan and execute the task, showing progress in real-time.

Examples:
  mix agent execute "setup python development environment"
  mix agent execute "install and configure nginx"
  mix agent execute "create a FastAPI project"`,
	Args: cobra.ExactArgs(1),
	Run:  runExecute,
}

type ExecuteRequest struct {
	Task        string `json:"task"`
	AutoConfirm bool   `json:"auto_confirm"`
}

type ExecuteResponse struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Steps   []Step `json:"steps,omitempty"`
	Error   string `json:"error,omitempty"`
}

type Step struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Output string `json:"output,omitempty"`
}

var autoConfirm bool

func init() {
	executeCmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "auto-confirm all actions")
}

func runExecute(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	task := args[0]

	fmt.Println(cyan("═══════════════════════════════════════"))
	fmt.Println(cyan("       MIXOS AI Task Execution         "))
	fmt.Println(cyan("═══════════════════════════════════════"))
	fmt.Println()
	fmt.Printf("%s Task: %s\n", cyan("→"), task)
	fmt.Println()

	// Send execute request
	reqBody := ExecuteRequest{
		Task:        task,
		AutoConfirm: autoConfirm,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("%s Failed to encode request: %v\n", red("✗"), err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Post(
		"http://localhost:8765/api/execute",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		fmt.Printf("%s Failed to connect to agent: %v\n", red("✗"), err)
		fmt.Println("Make sure the agent is running: mix agent start")
		os.Exit(1)
	}
	defer resp.Body.Close()

	var execResp ExecuteResponse
	if err := json.NewDecoder(resp.Body).Decode(&execResp); err != nil {
		fmt.Printf("%s Failed to decode response: %v\n", red("✗"), err)
		os.Exit(1)
	}

	if execResp.Error != "" {
		fmt.Printf("%s Error: %s\n", red("✗"), execResp.Error)
		os.Exit(1)
	}

	// Display results
	fmt.Printf("%s Task ID: %s\n", cyan("→"), execResp.TaskID)
	fmt.Println()

	if len(execResp.Steps) > 0 {
		fmt.Println(cyan("Execution Steps"))
		fmt.Println(strings.Repeat("─", 40))
		for i, step := range execResp.Steps {
			var statusIcon string
			switch step.Status {
			case "done":
				statusIcon = green("✓")
			case "failed":
				statusIcon = red("✗")
			case "skipped":
				statusIcon = yellow("○")
			default:
				statusIcon = cyan("→")
			}
			fmt.Printf("  %s [%d] %s\n", statusIcon, i+1, step.Name)
			if step.Output != "" {
				fmt.Printf("      %s\n", step.Output)
			}
		}
		fmt.Println()
	}

	// Final status
	switch execResp.Status {
	case "completed":
		fmt.Printf("%s %s\n", green("✓"), execResp.Message)
	case "failed":
		fmt.Printf("%s %s\n", red("✗"), execResp.Message)
		os.Exit(1)
	default:
		fmt.Printf("%s %s\n", yellow("→"), execResp.Message)
	}
}
