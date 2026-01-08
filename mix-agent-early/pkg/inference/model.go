package inference

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ModelManager manages AI models for early boot
type ModelManager struct {
	model       *LlamaModel
	config      ModelConfig
	systemPrompt string
	mu          sync.RWMutex
}

// NewModelManager creates a new model manager
func NewModelManager() *ModelManager {
	return &ModelManager{
		model: NewLlamaModel(),
		config: DefaultModelConfig(),
		systemPrompt: defaultSystemPrompt,
	}
}

const defaultSystemPrompt = `You are Mix Agent Early, an AI assistant for MIXOS GO early boot.
Your role is to help with:
- Hardware detection and module loading
- Root filesystem detection
- Boot troubleshooting
- Emergency recovery

Be concise and technical. Output commands when needed.
Format tool calls as: <tool>name</tool><params>{"key": "value"}</params>`

// ToolCall represents a parsed tool call
type ToolCall struct {
	Name   string                 `json:"name"`
	Params map[string]interface{} `json:"params"`
}

// Response represents a model response
type Response struct {
	Text      string     `json:"text"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// LoadModel loads the AI model
func (m *ModelManager) LoadModel(modelPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if model file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file not found: %s", modelPath)
	}

	m.config.Path = modelPath
	return m.model.Load(m.config)
}

// LoadModelWithConfig loads the model with custom configuration
func (m *ModelManager) LoadModelWithConfig(config ModelConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, err := os.Stat(config.Path); os.IsNotExist(err) {
		return fmt.Errorf("model file not found: %s", config.Path)
	}

	m.config = config
	return m.model.Load(config)
}

// UnloadModel unloads the current model
func (m *ModelManager) UnloadModel() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.model.Unload()
}

// IsReady returns whether the model is ready
func (m *ModelManager) IsReady() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.model.IsLoaded()
}

// SetSystemPrompt sets the system prompt
func (m *ModelManager) SetSystemPrompt(prompt string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.systemPrompt = prompt
}

// Generate generates a response for the given input
func (m *ModelManager) Generate(input string) (*Response, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.model.IsLoaded() {
		return m.mockGenerate(input)
	}

	// Build prompt with system prompt
	prompt := fmt.Sprintf("%s\n\nUser: %s\nAssistant:", m.systemPrompt, input)

	// Generate response
	text, err := m.model.Generate(prompt, DefaultGenerateConfig())
	if err != nil {
		return nil, err
	}

	// Parse response for tool calls
	response := &Response{
		Text:      text,
		ToolCalls: parseToolCalls(text),
	}

	return response, nil
}

// GenerateWithContext generates with conversation context
func (m *ModelManager) GenerateWithContext(messages []Message) (*Response, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.model.IsLoaded() {
		if len(messages) > 0 {
			return m.mockGenerate(messages[len(messages)-1].Content)
		}
		return m.mockGenerate("")
	}

	// Build prompt from messages
	var prompt strings.Builder
	prompt.WriteString(m.systemPrompt)
	prompt.WriteString("\n\n")

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			prompt.WriteString("User: ")
		case "assistant":
			prompt.WriteString("Assistant: ")
		case "system":
			continue // Already included
		case "tool":
			prompt.WriteString("Tool Result: ")
		}
		prompt.WriteString(msg.Content)
		prompt.WriteString("\n")
	}
	prompt.WriteString("Assistant:")

	// Generate
	text, err := m.model.Generate(prompt.String(), DefaultGenerateConfig())
	if err != nil {
		return nil, err
	}

	return &Response{
		Text:      text,
		ToolCalls: parseToolCalls(text),
	}, nil
}

// Message represents a conversation message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// mockGenerate provides responses when model is not loaded
func (m *ModelManager) mockGenerate(input string) (*Response, error) {
	inputLower := strings.ToLower(input)

	var text string
	var toolCalls []ToolCall

	switch {
	case strings.Contains(inputLower, "detect") && strings.Contains(inputLower, "hardware"):
		text = "I'll detect the hardware configuration."
		toolCalls = []ToolCall{{
			Name:   "detect_hardware",
			Params: map[string]interface{}{},
		}}

	case strings.Contains(inputLower, "load") && strings.Contains(inputLower, "module"):
		// Extract module name if present
		moduleName := "auto"
		words := strings.Fields(input)
		for i, w := range words {
			if w == "module" && i+1 < len(words) {
				moduleName = words[i+1]
				break
			}
		}
		text = fmt.Sprintf("Loading kernel module: %s", moduleName)
		toolCalls = []ToolCall{{
			Name:   "load_module",
			Params: map[string]interface{}{"module": moduleName},
		}}

	case strings.Contains(inputLower, "find") && strings.Contains(inputLower, "root"):
		text = "Searching for root filesystem..."
		toolCalls = []ToolCall{{
			Name:   "find_root",
			Params: map[string]interface{}{},
		}}

	case strings.Contains(inputLower, "mount"):
		text = "Mounting filesystem..."
		toolCalls = []ToolCall{{
			Name:   "mount_filesystem",
			Params: map[string]interface{}{},
		}}

	case strings.Contains(inputLower, "emergency") || strings.Contains(inputLower, "shell"):
		text = "Dropping to emergency shell for manual recovery."
		toolCalls = []ToolCall{{
			Name:   "emergency_shell",
			Params: map[string]interface{}{},
		}}

	case strings.Contains(inputLower, "status"):
		text = "System status: Early boot phase. Hardware detection pending."

	case strings.Contains(inputLower, "help"):
		text = `Available commands:
- detect hardware: Scan and identify hardware
- load module <name>: Load a kernel module
- find root: Search for root filesystem
- mount: Mount detected filesystems
- emergency shell: Drop to recovery shell
- status: Show current boot status`

	default:
		text = "I'm Mix Agent Early. I can help with hardware detection, module loading, and boot troubleshooting. Type 'help' for available commands."
	}

	return &Response{
		Text:      text,
		ToolCalls: toolCalls,
	}, nil
}

// parseToolCalls extracts tool calls from response text
func parseToolCalls(text string) []ToolCall {
	var calls []ToolCall

	// Look for <tool>name</tool><params>{...}</params> pattern
	for {
		toolStart := strings.Index(text, "<tool>")
		if toolStart == -1 {
			break
		}

		toolEnd := strings.Index(text[toolStart:], "</tool>")
		if toolEnd == -1 {
			break
		}
		toolEnd += toolStart

		toolName := text[toolStart+6 : toolEnd]

		// Look for params
		paramsStart := strings.Index(text[toolEnd:], "<params>")
		if paramsStart == -1 {
			calls = append(calls, ToolCall{Name: toolName, Params: map[string]interface{}{}})
			text = text[toolEnd+7:]
			continue
		}
		paramsStart += toolEnd

		paramsEnd := strings.Index(text[paramsStart:], "</params>")
		if paramsEnd == -1 {
			calls = append(calls, ToolCall{Name: toolName, Params: map[string]interface{}{}})
			text = text[toolEnd+7:]
			continue
		}
		paramsEnd += paramsStart

		paramsJSON := text[paramsStart+8 : paramsEnd]
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
			params = map[string]interface{}{}
		}

		calls = append(calls, ToolCall{Name: toolName, Params: params})
		text = text[paramsEnd+9:]
	}

	return calls
}

// FindModel searches for a model file in common locations
func FindModel() (string, error) {
	searchPaths := []string{
		"/opt/mixos/ai/model/mix-small-1.1b-q4.gguf",
		"/opt/mixos/ai/model/mix-early.gguf",
		"/boot/ai/model.gguf",
		"/initramfs/ai/model.gguf",
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Search in /opt/mixos/ai/model/ for any .gguf file
	modelDir := "/opt/mixos/ai/model"
	if entries, err := os.ReadDir(modelDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".gguf") {
				return filepath.Join(modelDir, entry.Name()), nil
			}
		}
	}

	return "", fmt.Errorf("no model file found")
}

// GetModelInfo returns information about the loaded model
func (m *ModelManager) GetModelInfo() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	info := m.model.GetInfo()
	info["system_prompt_length"] = len(m.systemPrompt)
	return info
}
