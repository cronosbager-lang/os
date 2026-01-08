package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for packages",
	Long: `Search for packages in the repository.

Examples:
  mix search docker
  mix search python`,
	Args: cobra.ExactArgs(1),
	Run:  runSearch,
}

var infoCmd = &cobra.Command{
	Use:   "info [package]",
	Short: "Show package information",
	Long: `Display detailed information about a package.

Examples:
  mix info docker
  mix info python`,
	Args: cobra.ExactArgs(1),
	Run:  runInfo,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed packages",
	Long:  `List all installed packages on the system.`,
	Run:   runList,
}

func init() {
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(listCmd)
}

func runSearch(cmd *cobra.Command, args []string) {
	mixPkg := exec.Command("mix-pkg", "search", args[0])
	mixPkg.Stdout = os.Stdout
	mixPkg.Stderr = os.Stderr

	if err := mixPkg.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Search failed: %v\n", err)
		os.Exit(1)
	}
}

func runInfo(cmd *cobra.Command, args []string) {
	mixPkg := exec.Command("mix-pkg", "info", args[0])
	mixPkg.Stdout = os.Stdout
	mixPkg.Stderr = os.Stderr

	if err := mixPkg.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get package info: %v\n", err)
		os.Exit(1)
	}
}

func runList(cmd *cobra.Command, args []string) {
	mixPkg := exec.Command("mix-pkg", "list")
	mixPkg.Stdout = os.Stdout
	mixPkg.Stderr = os.Stderr

	if err := mixPkg.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list packages: %v\n", err)
		os.Exit(1)
	}
}
