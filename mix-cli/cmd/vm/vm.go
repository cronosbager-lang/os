package vm

import (
	"github.com/spf13/cobra"
)

var VMCmd = &cobra.Command{
	Use:   "vm",
	Short: "Virtual machine management",
	Long: `Manage virtual machines.

Commands:
  create    Create a new VM
  start     Start a VM
  stop      Stop a VM
  list      List all VMs
  destroy   Destroy a VM`,
}

func init() {
	VMCmd.AddCommand(createCmd)
	VMCmd.AddCommand(startCmd)
	VMCmd.AddCommand(stopCmd)
	VMCmd.AddCommand(listCmd)
	VMCmd.AddCommand(destroyCmd)
}
