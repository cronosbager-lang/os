package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install [packages...]",
	Short: "Install packages",
	Long: `Install one or more packages using mix-pkg.

Examples:
  mix install docker
  mix install python nodejs go
  mix install --force nginx`,
	Args: cobra.MinimumNArgs(1),
	Run:  runInstall,
}

var (
	forceInstall bool
)

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVar(&forceInstall, "force", false, "force reinstallation")
}

func runInstall(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	fmt.Printf("%s Installing packages: %v\n", cyan("→"), args)

	// Build mix-pkg command
	mixPkgArgs := []string{"install"}
	if forceInstall {
		mixPkgArgs = append(mixPkgArgs, "--force")
	}
	if yes {
		mixPkgArgs = append(mixPkgArgs, "-y")
	}
	mixPkgArgs = append(mixPkgArgs, args...)

	// Execute mix-pkg
	mixPkg := exec.Command("mix-pkg", mixPkgArgs...)
	mixPkg.Stdout = os.Stdout
	mixPkg.Stderr = os.Stderr
	mixPkg.Stdin = os.Stdin

	if err := mixPkg.Run(); err != nil {
		fmt.Printf("%s Installation failed: %v\n", red("✗"), err)
		os.Exit(1)
	}

	fmt.Printf("%s Packages installed successfully\n", green("✓"))
}
