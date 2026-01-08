package agent

import (
	"github.com/spf13/cobra"
)

var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "AI agent management",
	Long: `Manage the MIXOS AI agent.

The AI agent provides intelligent assistance for system operations,
development tasks, and autonomous management.

Commands:
  start     Start the AI agent service
  stop      Stop the AI agent service
  status    Show agent status
  restart   Restart the agent
  chat      Interactive chat with AI
  execute   Execute an autonomous task`,
}

func init() {
	AgentCmd.AddCommand(startCmd)
	AgentCmd.AddCommand(stopCmd)
	AgentCmd.AddCommand(statusCmd)
	AgentCmd.AddCommand(restartCmd)
	AgentCmd.AddCommand(chatCmd)
	AgentCmd.AddCommand(executeCmd)
	AgentCmd.AddCommand(configCmd)
}
