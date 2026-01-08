package agent

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the AI agent",
	Long:  `Start the MIXOS AI agent service.`,
	Run:   runStart,
}

func runStart(cmd *cobra.Command, args []string) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("%s Starting AI agent...\n", cyan("→"))

	systemctl := exec.Command("systemctl", "start", "mixos-agent")
	systemctl.Stdout = os.Stdout
	systemctl.Stderr = os.Stderr

	if err := systemctl.Run(); err != nil {
		fmt.Printf("%s Failed to start agent: %v\n", red("✗"), err)
		os.Exit(1)
	}

	fmt.Printf("%s AI agent started\n", green("✓"))
}
