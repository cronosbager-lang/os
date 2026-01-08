package container

import (
	"github.com/spf13/cobra"
)

var ContainerCmd = &cobra.Command{
	Use:     "container",
	Aliases: []string{"docker"},
	Short:   "Container management",
	Long: `Manage Docker containers.

Commands:
  list    List running containers
  logs    Show container logs
  exec    Execute command in container`,
}

func init() {
	ContainerCmd.AddCommand(listCmd)
	ContainerCmd.AddCommand(logsCmd)
	ContainerCmd.AddCommand(execCmd)
}
