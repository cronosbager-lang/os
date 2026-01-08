package agent

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the AI agent",
	Long:  `Stop the MIXOS AI agent service.`,
	Run:   runStop,
}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the AI agent",
	Long:  `Restart the MIXOS AI agent service.`,
	Run:   runRestart,
}

func runStop(cmd *cobra.Command, args []string) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("%s Stopping AI agent...\n", cyan("→"))

	systemctl := exec.Command("systemctl", "stop", "mixos-agent")
	systemctl.Stdout = os.Stdout
	systemctl.Stderr = os.Stderr

	if err := systemctl.Run(); err != nil {
		fmt.Printf("%s Failed to stop agent: %v\n", red("✗"), err)
		os.Exit(1)
	}

	fmt.Printf("%s AI agent stopped\n", green("✓"))
}

func runRestart(cmd *cobra.Command, args []string) {
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("%s Restarting AI agent...\n", cyan("→"))

	systemctl := exec.Command("systemctl", "restart", "mixos-agent")
	systemctl.Stdout = os.Stdout
	systemctl.Stderr = os.Stderr

	if err := systemctl.Run(); err != nil {
		fmt.Printf("%s Failed to restart agent: %v\n", red("✗"), err)
		os.Exit(1)
	}

	fmt.Printf("%s AI agent restarted\n", green("✓"))
}
