package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove [packages...]",
	Aliases: []string{"uninstall", "rm"},
	Short:   "Remove packages",
	Long: `Remove one or more installed packages.

Examples:
  mix remove docker
  mix remove python nodejs`,
	Args: cobra.MinimumNArgs(1),
	Run:  runRemove,
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	fmt.Printf("%s Removing packages: %v\n", cyan("→"), args)

	mixPkgArgs := []string{"remove"}
	if yes {
		mixPkgArgs = append(mixPkgArgs, "-y")
	}
	mixPkgArgs = append(mixPkgArgs, args...)

	mixPkg := exec.Command("mix-pkg", mixPkgArgs...)
	mixPkg.Stdout = os.Stdout
	mixPkg.Stderr = os.Stderr
	mixPkg.Stdin = os.Stdin

	if err := mixPkg.Run(); err != nil {
		fmt.Printf("%s Removal failed: %v\n", red("✗"), err)
		os.Exit(1)
	}

	fmt.Printf("%s Packages removed successfully\n", green("✓"))
}
