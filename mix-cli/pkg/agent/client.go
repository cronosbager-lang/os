package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client for communicating with the MIXOS AI Agent
type Client struct {
	baseURL    string
	httpClient *http.Client
	sessionID  string
}

// NewClient creates a new agent client
func NewClient() *Client {
	return &Client{
		baseURL: "http://localhost:8765",
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// NewClientWithURL creates a client with custom URL
func NewClientWithURL(url string) *Client {
	return &Client{
		baseURL: url,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// SetSessionID sets the session ID for conversation continuity
func (c *Client) SetSessionID(id string) {
	c.sessionID = id
}

// ChatRequest represents a chat request
type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
	Stream    bool   `json:"stream"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	Response  string `json:"response"`
	SessionID string `json:"session_id"`
	Error     string `json:"error,omitempty"`
}

// ExecuteRequest represents a task execution request
type ExecuteRequest struct {
	Task        string `json:"task"`
	AutoConfirm bool   `json:"auto_confirm"`
}

// ExecuteResponse represents a task execution response
type ExecuteResponse struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Steps   []Step `json:"steps,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Step represents an execution step
type Step struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Output string `json:"output,omitempty"`
}

// StatusResponse represents agent status
type StatusResponse struct {
	Status      string `json:"status"`
	Model       string `json:"model"`
	Uptime      string `json:"uptime"`
	MemoryUsage string `json:"memory_usage"`
	RequestsOK  int    `json:"requests_ok"`
}

// Chat sends a message to the agent
func (c *Client) Chat(message string) (*ChatResponse, error) {
	req := ChatRequest{
		Message:   message,
		SessionID: c.sessionID,
		Stream:    false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/chat",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent returned status %d", resp.StatusCode)
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Update session ID
	if chatResp.SessionID != "" {
		c.sessionID = chatResp.SessionID
	}

	return &chatResp, nil
}

// Execute runs an autonomous task
func (c *Client) Execute(task string, autoConfirm bool) (*ExecuteResponse, error) {
	req := ExecuteRequest{
		Task:        task,
		AutoConfirm: autoConfirm,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/execute",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent: %w", err)
	}
	defer resp.Body.Close()

	var execResp ExecuteResponse
	if err := json.NewDecoder(resp.Body).Decode(&execResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &execResp, nil
}

// Status gets the agent status
func (c *Client) Status() (*StatusResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/status")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent returned status %d", resp.StatusCode)
	}

	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// Health checks if the agent is healthy
func (c *Client) Health() (bool, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/health")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// StreamChat sends a message and streams the response
func (c *Client) StreamChat(message string, callback func(chunk string)) error {
	req := ChatRequest{
		Message:   message,
		SessionID: c.sessionID,
		Stream:    true,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/chat/stream",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to agent: %w", err)
	}
	defer resp.Body.Close()

	// Read streaming response
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			callback(string(buf[:n]))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading stream: %w", err)
		}
	}

	return nil
}

// GetTools returns available tools
func (c *Client) GetTools() ([]string, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/tools")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent: %w", err)
	}
	defer resp.Body.Close()

	var tools struct {
		Tools []string `json:"tools"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tools); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return tools.Tools, nil
}

// GetConfig returns agent configuration
func (c *Client) GetConfig() (map[string]interface{}, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/config")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent: %w", err)
	}
	defer resp.Body.Close()

	var config map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return config, nil
}
