package agent

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure AI agent settings",
	Long:  `View and modify AI agent configuration.`,
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	Run:   runConfigGet,
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	Run:   runConfigSet,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration",
	Run:   runConfigList,
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListCmd)
}

func runConfigGet(cmd *cobra.Command, args []string) {
	loadAgentConfig()
	key := args[0]
	value := viper.Get(key)
	if value == nil {
		fmt.Printf("Key '%s' not found\n", key)
		os.Exit(1)
	}
	fmt.Printf("%s = %v\n", key, value)
}

func runConfigSet(cmd *cobra.Command, args []string) {
	loadAgentConfig()
	key := args[0]
	value := args[1]

	viper.Set(key, value)
	if err := viper.WriteConfig(); err != nil {
		fmt.Printf("Failed to write config: %v\n", err)
		os.Exit(1)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Set %s = %s\n", green("✓"), key, value)
}

func runConfigList(cmd *cobra.Command, args []string) {
	loadAgentConfig()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Println(cyan("AI Agent Configuration"))
	fmt.Println("───────────────────────────────────────")

	settings := viper.AllSettings()
	printSettings("", settings)
}

func loadAgentConfig() {
	viper.SetConfigName("agent")
	viper.SetConfigType("toml")
	viper.AddConfigPath("/etc/mixos")
	viper.AddConfigPath("$HOME/.config/mixos")

	if err := viper.ReadInConfig(); err != nil {
		// Use defaults if no config file
		viper.SetDefault("agent.name", "Mix Agent")
		viper.SetDefault("agent.enabled", true)
		viper.SetDefault("model.path", "/opt/mixos/ai/model/mix-small-1.1b-q4.gguf")
		viper.SetDefault("model.context_length", 4096)
		viper.SetDefault("model.temperature", 0.7)
		viper.SetDefault("inference.threads", 4)
	}
}

func printSettings(prefix string, settings map[string]interface{}) {
	for key, value := range settings {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			printSettings(fullKey, v)
		default:
			fmt.Printf("  %s = %v\n", fullKey, value)
		}
	}
}
