package main

import (
	"fmt"
	"os"

	"github.com/mixos/mix-agent-early/cmd"
)

func main() {
	if len(os.Args) < 2 {
		cmd.RunInit()
		return
	}

	switch os.Args[1] {
	case "init":
		cmd.RunInit()
	case "detect":
		cmd.RunDetect()
	case "analyze":
		cmd.RunAnalyze()
	case "emergency":
		cmd.RunEmergency()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		fmt.Fprintln(os.Stderr, "Usage: mix-agent-early [init|detect|analyze|emergency]")
		os.Exit(1)
	}
}
