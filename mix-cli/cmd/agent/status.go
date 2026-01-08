package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show AI agent status",
	Long:  `Display the current status of the MIXOS AI agent.`,
	Run:   runStatus,
}

type AgentStatus struct {
	Status      string `json:"status"`
	Model       string `json:"model"`
	Uptime      string `json:"uptime"`
	MemoryUsage string `json:"memory_usage"`
	RequestsOK  int    `json:"requests_ok"`
}

func runStatus(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Println(cyan("═══════════════════════════════════════"))
	fmt.Println(cyan("         MIXOS AI Agent Status         "))
	fmt.Println(cyan("═══════════════════════════════════════"))
	fmt.Println()

	// Check systemd service status
	out, err := exec.Command("systemctl", "is-active", "mixos-agent").Output()
	serviceStatus := strings.TrimSpace(string(out))

	fmt.Println(cyan("Service"))
	fmt.Println(strings.Repeat("─", 40))
	if err == nil && serviceStatus == "active" {
		fmt.Printf("  Status:       %s\n", green("● Running"))
	} else {
		fmt.Printf("  Status:       %s\n", yellow("○ Stopped"))
		fmt.Println()
		fmt.Println("Start the agent with: mix agent start")
		return
	}

	// Try to get detailed status from agent API
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:8765/api/status")
	if err != nil {
		fmt.Printf("  API:          %s\n", yellow("Not responding"))
		fmt.Println()
		return
	}
	defer resp.Body.Close()

	var status AgentStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		fmt.Printf("  API:          %s\n", yellow("Invalid response"))
		fmt.Println()
		return
	}

	fmt.Printf("  API:          %s\n", green("● Connected"))
	fmt.Println()

	fmt.Println(cyan("Model"))
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("  Name:         %s\n", status.Model)
	fmt.Printf("  Memory:       %s\n", status.MemoryUsage)
	fmt.Println()

	fmt.Println(cyan("Statistics"))
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("  Uptime:       %s\n", status.Uptime)
	fmt.Printf("  Requests:     %d\n", status.RequestsOK)
	fmt.Println()
}
