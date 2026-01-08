package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update package database",
	Long:  `Synchronize the package database with remote repositories.`,
	Run:   runUpdate,
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade all packages",
	Long:  `Upgrade all installed packages to their latest versions.`,
	Run:   runUpgrade,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(upgradeCmd)
}

func runUpdate(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	fmt.Printf("%s Updating package database...\n", cyan("→"))

	mixPkg := exec.Command("mix-pkg", "update")
	mixPkg.Stdout = os.Stdout
	mixPkg.Stderr = os.Stderr

	if err := mixPkg.Run(); err != nil {
		fmt.Printf("%s Update failed: %v\n", red("✗"), err)
		os.Exit(1)
	}

	fmt.Printf("%s Package database updated\n", green("✓"))
}

func runUpgrade(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	fmt.Printf("%s Upgrading system packages...\n", cyan("→"))

	mixPkgArgs := []string{"upgrade"}
	if yes {
		mixPkgArgs = append(mixPkgArgs, "-y")
	}

	mixPkg := exec.Command("mix-pkg", mixPkgArgs...)
	mixPkg.Stdout = os.Stdout
	mixPkg.Stderr = os.Stderr
	mixPkg.Stdin = os.Stdin

	if err := mixPkg.Run(); err != nil {
		fmt.Printf("%s Upgrade failed: %v\n", red("✗"), err)
		os.Exit(1)
	}

	fmt.Printf("%s System upgraded successfully\n", green("✓"))
}
