// Package ipc provides IPC client for mix-cli to communicate with MixOS services
package ipc

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultSocketPath = "/run/mixos/ipc.sock"
	DefaultTimeout    = 30 * time.Second
)

// MessageType represents IPC message types
type MessageType string

const (
	TypeRequest  MessageType = "REQUEST"
	TypeResponse MessageType = "RESPONSE"
	TypeEvent    MessageType = "EVENT"
	TypeStream   MessageType = "STREAM"
)

// Message represents an IPC message
type Message struct {
	Version   int         `json:"version"`
	MsgType   MessageType `json:"msg_type"`
	MsgID     uint64      `json:"msg_id"`
	Source    string      `json:"source"`
	Target    string      `json:"target"`
	Method    string      `json:"method"`
	Payload   string      `json:"payload"`
	Timestamp int64       `json:"timestamp"`
	Error     string      `json:"error,omitempty"`
}

// Client is an IPC client for communicating with MixOS services
type Client struct {
	socketPath string
	conn       net.Conn
	msgCounter uint64
	mu         sync.Mutex
	timeout    time.Duration
}

// NewClient creates a new IPC client
func NewClient(socketPath string) *Client {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}
	return &Client{
		socketPath: socketPath,
		timeout:    DefaultTimeout,
	}
}

// Connect establishes connection to the IPC broker
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", c.socketPath, err)
	}
	c.conn = conn

	// Register as mix-cli
	return c.register()
}

func (c *Client) register() error {
	msg := &Message{
		Version:   1,
		MsgType:   TypeRequest,
		MsgID:     c.nextMsgID(),
		Source:    "mix-cli",
		Target:    "broker",
		Method:    "register",
		Payload:   `{"service": "mix-cli", "type": "client"}`,
		Timestamp: time.Now().UnixMilli(),
	}

	if err := c.sendMessage(msg); err != nil {
		return err
	}

	_, err := c.readMessage()
	return err
}

func (c *Client) nextMsgID() uint64 {
	return atomic.AddUint64(&c.msgCounter, 1)
}

// Call makes an RPC call to a service
func (c *Client) Call(target, method string, payload interface{}) (*Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := &Message{
		Version:   1,
		MsgType:   TypeRequest,
		MsgID:     c.nextMsgID(),
		Source:    "mix-cli",
		Target:    target,
		Method:    method,
		Payload:   string(payloadBytes),
		Timestamp: time.Now().UnixMilli(),
	}

	if err := c.sendMessage(msg); err != nil {
		return nil, err
	}

	c.conn.SetReadDeadline(time.Now().Add(c.timeout))
	return c.readMessage()
}

func (c *Client) sendMessage(msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Write length prefix
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))
	if _, err := c.conn.Write(lenBuf); err != nil {
		return err
	}

	// Write message
	_, err = c.conn.Write(data)
	return err
}

func (c *Client) readMessage() (*Message, error) {
	// Read length prefix
	lenBuf := make([]byte, 4)
	if _, err := c.conn.Read(lenBuf); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf)

	// Read message
	msgBuf := make([]byte, length)
	if _, err := c.conn.Read(msgBuf); err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(msgBuf, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

// Close closes the connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SetTimeout sets the timeout for RPC calls
func (c *Client) SetTimeout(d time.Duration) {
	c.timeout = d
}

// Package management helpers

// PackageRequest represents a package operation request
type PackageRequest struct {
	Action   string            `json:"action"`
	Packages []string          `json:"packages"`
	Options  map[string]string `json:"options,omitempty"`
}

// PackageResponse represents a package operation response
type PackageResponse struct {
	Success  bool          `json:"success"`
	Error    string        `json:"error,omitempty"`
	Packages []PackageInfo `json:"packages,omitempty"`
}

// PackageInfo represents package information
type PackageInfo struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies"`
	Size         int64    `json:"size"`
	Hash         string   `json:"hash"`
	Installed    bool     `json:"installed"`
}

// InstallPackages installs packages via pkgmgr service
func (c *Client) InstallPackages(packages []string) (*PackageResponse, error) {
	req := PackageRequest{
		Action:   "install",
		Packages: packages,
	}

	resp, err := c.Call("pkgmgr", "package", req)
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, fmt.Errorf(resp.Error)
	}

	var pkgResp PackageResponse
	if err := json.Unmarshal([]byte(resp.Payload), &pkgResp); err != nil {
		return nil, err
	}

	return &pkgResp, nil
}

// RemovePackages removes packages via pkgmgr service
func (c *Client) RemovePackages(packages []string) (*PackageResponse, error) {
	req := PackageRequest{
		Action:   "remove",
		Packages: packages,
	}

	resp, err := c.Call("pkgmgr", "package", req)
	if err != nil {
		return nil, err
	}

	var pkgResp PackageResponse
	if err := json.Unmarshal([]byte(resp.Payload), &pkgResp); err != nil {
		return nil, err
	}

	return &pkgResp, nil
}

// QueryPackages queries package information
func (c *Client) QueryPackages(packages []string) (*PackageResponse, error) {
	req := PackageRequest{
		Action:   "query",
		Packages: packages,
	}

	resp, err := c.Call("pkgmgr", "package", req)
	if err != nil {
		return nil, err
	}

	var pkgResp PackageResponse
	if err := json.Unmarshal([]byte(resp.Payload), &pkgResp); err != nil {
		return nil, err
	}

	return &pkgResp, nil
}

// ListPackages lists all installed packages
func (c *Client) ListPackages() (*PackageResponse, error) {
	req := PackageRequest{
		Action: "list",
	}

	resp, err := c.Call("pkgmgr", "package", req)
	if err != nil {
		return nil, err
	}

	var pkgResp PackageResponse
	if err := json.Unmarshal([]byte(resp.Payload), &pkgResp); err != nil {
		return nil, err
	}

	return &pkgResp, nil
}

// Build helpers

// BuildRequest represents a build request
type BuildRequest struct {
	PackageName string            `json:"package_name"`
	SourcePath  string            `json:"source_path"`
	OutputPath  string            `json:"output_path"`
	Env         map[string]string `json:"env,omitempty"`
	BuildArgs   []string          `json:"build_args,omitempty"`
}

// BuildResponse represents a build response
type BuildResponse struct {
	Success      bool     `json:"success"`
	Error        string   `json:"error,omitempty"`
	ArtifactPath string   `json:"artifact_path"`
	ArtifactHash string   `json:"artifact_hash"`
	Logs         []string `json:"logs"`
}

// Build executes a build via builder service
func (c *Client) Build(req BuildRequest) (*BuildResponse, error) {
	resp, err := c.Call("builder", "build", req)
	if err != nil {
		return nil, err
	}

	var buildResp BuildResponse
	if err := json.Unmarshal([]byte(resp.Payload), &buildResp); err != nil {
		return nil, err
	}

	return &buildResp, nil
}

// Resolver helpers

// ResolveRequest represents a dependency resolution request
type ResolveRequest struct {
	Packages        []string `json:"packages"`
	IncludeOptional bool     `json:"include_optional"`
}

// ResolveResponse represents a dependency resolution response
type ResolveResponse struct {
	Success      bool             `json:"success"`
	Error        string           `json:"error,omitempty"`
	Graph        []DependencyNode `json:"graph"`
	InstallOrder []string         `json:"install_order"`
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Dependencies []string `json:"dependencies"`
	IsInstalled  bool     `json:"is_installed"`
}

// ResolveDependencies resolves package dependencies
func (c *Client) ResolveDependencies(packages []string, includeOptional bool) (*ResolveResponse, error) {
	req := ResolveRequest{
		Packages:        packages,
		IncludeOptional: includeOptional,
	}

	resp, err := c.Call("resolver", "resolve", req)
	if err != nil {
		return nil, err
	}

	var resolveResp ResolveResponse
	if err := json.Unmarshal([]byte(resp.Payload), &resolveResp); err != nil {
		return nil, err
	}

	return &resolveResp, nil
}

// Agent helpers

// AgentRequest represents an agent request
type AgentRequest struct {
	Action  string            `json:"action"`
	Prompt  string            `json:"prompt,omitempty"`
	Context map[string]string `json:"context,omitempty"`
}

// AgentResponse represents an agent response
type AgentResponse struct {
	Success  bool     `json:"success"`
	Error    string   `json:"error,omitempty"`
	Response string   `json:"response"`
	Actions  []string `json:"actions"`
}

// AgentChat sends a chat message to the agent
func (c *Client) AgentChat(prompt string, context map[string]string) (*AgentResponse, error) {
	req := AgentRequest{
		Action:  "chat",
		Prompt:  prompt,
		Context: context,
	}

	resp, err := c.Call("agent", "chat", req)
	if err != nil {
		return nil, err
	}

	var agentResp AgentResponse
	if err := json.Unmarshal([]byte(resp.Payload), &agentResp); err != nil {
		return nil, err
	}

	return &agentResp, nil
}

// AgentStatus gets the agent status
func (c *Client) AgentStatus() (*AgentResponse, error) {
	req := AgentRequest{
		Action: "status",
	}

	resp, err := c.Call("agent", "status", req)
	if err != nil {
		return nil, err
	}

	var agentResp AgentResponse
	if err := json.Unmarshal([]byte(resp.Payload), &agentResp); err != nil {
		return nil, err
	}

	return &agentResp, nil
}
