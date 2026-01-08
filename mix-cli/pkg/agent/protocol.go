package agent

import (
	"encoding/json"
	"fmt"
)

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeChat     MessageType = "chat"
	MessageTypeExecute  MessageType = "execute"
	MessageTypeStatus   MessageType = "status"
	MessageTypeError    MessageType = "error"
	MessageTypeProgress MessageType = "progress"
	MessageTypeComplete MessageType = "complete"
)

// Message represents a protocol message
type Message struct {
	Type      MessageType     `json:"type"`
	ID        string          `json:"id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp int64           `json:"timestamp,omitempty"`
}

// ProgressPayload represents progress information
type ProgressPayload struct {
	TaskID   string  `json:"task_id"`
	Step     int     `json:"step"`
	Total    int     `json:"total"`
	Message  string  `json:"message"`
	Progress float64 `json:"progress"`
}

// ErrorPayload represents an error
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ToolCall represents a tool call from the agent
type ToolCall struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

// ParseMessage parses a JSON message
func ParseMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}
	return &msg, nil
}

// NewChatMessage creates a new chat message
func NewChatMessage(id string, content string) *Message {
	payload, _ := json.Marshal(map[string]string{"content": content})
	return &Message{
		Type:    MessageTypeChat,
		ID:      id,
		Payload: payload,
	}
}

// NewProgressMessage creates a new progress message
func NewProgressMessage(taskID string, step, total int, message string, progress float64) *Message {
	payload, _ := json.Marshal(ProgressPayload{
		TaskID:   taskID,
		Step:     step,
		Total:    total,
		Message:  message,
		Progress: progress,
	})
	return &Message{
		Type:    MessageTypeProgress,
		Payload: payload,
	}
}

// NewErrorMessage creates a new error message
func NewErrorMessage(code, message, details string) *Message {
	payload, _ := json.Marshal(ErrorPayload{
		Code:    code,
		Message: message,
		Details: details,
	})
	return &Message{
		Type:    MessageTypeError,
		Payload: payload,
	}
}

// GetChatContent extracts chat content from message
func (m *Message) GetChatContent() (string, error) {
	if m.Type != MessageTypeChat {
		return "", fmt.Errorf("not a chat message")
	}

	var content struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(m.Payload, &content); err != nil {
		return "", err
	}
	return content.Content, nil
}

// GetProgress extracts progress from message
func (m *Message) GetProgress() (*ProgressPayload, error) {
	if m.Type != MessageTypeProgress {
		return nil, fmt.Errorf("not a progress message")
	}

	var progress ProgressPayload
	if err := json.Unmarshal(m.Payload, &progress); err != nil {
		return nil, err
	}
	return &progress, nil
}

// GetError extracts error from message
func (m *Message) GetError() (*ErrorPayload, error) {
	if m.Type != MessageTypeError {
		return nil, fmt.Errorf("not an error message")
	}

	var errPayload ErrorPayload
	if err := json.Unmarshal(m.Payload, &errPayload); err != nil {
		return nil, err
	}
	return &errPayload, nil
}

// Serialize converts message to JSON
func (m *Message) Serialize() ([]byte, error) {
	return json.Marshal(m)
}
