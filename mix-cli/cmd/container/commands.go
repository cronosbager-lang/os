package container

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var (
	follow bool
	tail   string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List running containers",
	Run:   runList,
}

var logsCmd = &cobra.Command{
	Use:   "logs [container]",
	Short: "Show container logs",
	Args:  cobra.ExactArgs(1),
	Run:   runLogs,
}

var execCmd = &cobra.Command{
	Use:   "exec [container] [command...]",
	Short: "Execute command in container",
	Args:  cobra.MinimumNArgs(2),
	Run:   runExec,
}

func init() {
	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow log output")
	logsCmd.Flags().StringVar(&tail, "tail", "100", "number of lines to show")
}

func runList(cmd *cobra.Command, args []string) {
	docker := exec.Command("docker", "ps", "--format",
		"table {{.ID}}\t{{.Image}}\t{{.Status}}\t{{.Names}}")
	docker.Stdout = os.Stdout
	docker.Stderr = os.Stderr

	if err := docker.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list containers: %v\n", err)
		os.Exit(1)
	}
}

func runLogs(cmd *cobra.Command, args []string) {
	container := args[0]

	dockerArgs := []string{"logs"}
	if follow {
		dockerArgs = append(dockerArgs, "-f")
	}
	dockerArgs = append(dockerArgs, "--tail", tail, container)

	docker := exec.Command("docker", dockerArgs...)
	docker.Stdout = os.Stdout
	docker.Stderr = os.Stderr

	if err := docker.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get logs: %v\n", err)
		os.Exit(1)
	}
}

func runExec(cmd *cobra.Command, args []string) {
	container := args[0]
	command := args[1:]

	dockerArgs := []string{"exec", "-it", container}
	dockerArgs = append(dockerArgs, command...)

	docker := exec.Command("docker", dockerArgs...)
	docker.Stdout = os.Stdout
	docker.Stderr = os.Stderr
	docker.Stdin = os.Stdin

	if err := docker.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to exec: %v\n", err)
		os.Exit(1)
	}
}
