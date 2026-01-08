package dev

import (
	"github.com/spf13/cobra"
)

var DevCmd = &cobra.Command{
	Use:   "dev",
	Short: "Developer tools",
	Long: `Developer environment management.

Commands:
  setup    Setup development environment
  init     Initialize a new project
  tools    Install development tools`,
}

func init() {
	DevCmd.AddCommand(setupCmd)
	DevCmd.AddCommand(initCmd)
	DevCmd.AddCommand(toolsCmd)
}
