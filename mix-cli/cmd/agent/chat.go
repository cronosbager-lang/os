package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Interactive chat with AI agent",
	Long: `Start an interactive chat session with the MIXOS AI agent.

The agent can help with:
  - System administration tasks
  - Development environment setup
  - Package management
  - Troubleshooting

Type 'exit' or 'quit' to end the session.`,
	Run: runChat,
}

type ChatRequest struct {
	Message string `json:"message"`
	Stream  bool   `json:"stream"`
}

type ChatResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

func runChat(cmd *cobra.Command, args []string) {
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Println(cyan("═══════════════════════════════════════"))
	fmt.Println(cyan("       MIXOS AI Agent Chat             "))
	fmt.Println(cyan("═══════════════════════════════════════"))
	fmt.Println()
	fmt.Println("Type your message and press Enter.")
	fmt.Println("Type 'exit' or 'quit' to end the session.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(green("You: "))
		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println()
				break
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println(yellow("Goodbye!"))
			break
		}

		response, err := sendChatMessage(input)
		if err != nil {
			fmt.Printf("%s Error: %v\n", color.RedString("✗"), err)
			continue
		}

		fmt.Printf("%s %s\n\n", cyan("AI:"), response)
	}
}

func sendChatMessage(message string) (string, error) {
	reqBody := ChatRequest{
		Message: message,
		Stream:  false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to encode request: %w", err)
	}

	resp, err := http.Post(
		"http://localhost:8765/api/chat",
		"application/json",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return "", fmt.Errorf("failed to connect to agent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("agent returned status %d", resp.StatusCode)
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if chatResp.Error != "" {
		return "", fmt.Errorf("agent error: %s", chatResp.Error)
	}

	return chatResp.Response, nil
}
