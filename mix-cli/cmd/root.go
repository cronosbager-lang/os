package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/mixos/mix-cli/cmd/agent"
	"github.com/mixos/mix-cli/cmd/container"
	"github.com/mixos/mix-cli/cmd/dev"
	"github.com/mixos/mix-cli/cmd/vm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
	quiet   bool
	yes     bool
)

var rootCmd = &cobra.Command{
	Use:   "mix",
	Short: "MIXOS command-line interface",
	Long: `mix is the command-line interface for MIXOS GO.

It provides commands for:
  - System management (status, update, upgrade)
  - Package operations (install, remove, search)
  - AI agent control (start, stop, chat, execute)
  - VM and container management
  - Developer tools

Examples:
  mix status                    Show system status
  mix install docker            Install a package
  mix agent chat                Chat with AI agent
  mix agent execute "setup go"  Execute AI task`,
	Version: "1.0.0",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: /etc/mixos/cli.toml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress output")
	rootCmd.PersistentFlags().BoolVarP(&yes, "yes", "y", false, "auto-confirm prompts")

	rootCmd.SetVersionTemplate(versionTemplate())

	// Add subcommands
	rootCmd.AddCommand(agent.AgentCmd)
	rootCmd.AddCommand(vm.VMCmd)
	rootCmd.AddCommand(container.ContainerCmd)
	rootCmd.AddCommand(dev.DevCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("/etc/mixos")
		viper.AddConfigPath("$HOME/.config/mixos")
		viper.SetConfigName("cli")
		viper.SetConfigType("toml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
		}
	}
}

func versionTemplate() string {
	cyan := color.New(color.FgCyan).SprintFunc()
	return fmt.Sprintf(`%s
Version:    {{.Version}}
OS:         MIXOS GO
`, cyan("mix - MIXOS CLI"))
}

func printSuccess(msg string) {
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s %s\n", green("✓"), msg)
}

func printError(msg string) {
	red := color.New(color.FgRed).SprintFunc()
	fmt.Printf("%s %s\n", red("✗"), msg)
}

func printInfo(msg string) {
	blue := color.New(color.FgBlue).SprintFunc()
	fmt.Printf("%s %s\n", blue("→"), msg)
}
