package ipc

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	pb "mixos.dev/init/pkg/ipc/proto"
)

// ClientConfig contains IPC client configuration
type ClientConfig struct {
	SocketPath       string
	ServiceName      string
	ConnectTimeout   time.Duration
	RequestTimeout   time.Duration
	ReconnectDelay   time.Duration
	MaxReconnectDelay time.Duration
	MaxRetries       int
	HeartbeatInterval time.Duration
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig(serviceName string) ClientConfig {
	return ClientConfig{
		SocketPath:        "/run/mixos/ipc.sock",
		ServiceName:       serviceName,
		ConnectTimeout:    5 * time.Second,
		RequestTimeout:    30 * time.Second,
		ReconnectDelay:    1 * time.Second,
		MaxReconnectDelay: 30 * time.Second,
		MaxRetries:        3,
		HeartbeatInterval: 30 * time.Second,
	}
}

// IPCClient is a client for connecting to the MixOS IPC server
type IPCClient struct {
	config     ClientConfig
	conn       net.Conn
	mu         sync.RWMutex
	connected  bool
	msgCounter uint64
	
	// Pending requests
	pending    map[uint64]chan *pb.IPCMessage
	pendingMu  sync.RWMutex
	
	// Handlers for incoming requests
	handlers   map[string]ClientHandler
	handlersMu sync.RWMutex
	
	// Event handlers
	eventHandlers map[string]EventHandler
	eventMu       sync.RWMutex
	
	// Context for shutdown
	ctx        context.Context
	cancel     context.CancelFunc
	
	// Reconnection state
	reconnecting bool
	reconnectMu  sync.Mutex
}

// ClientHandler handles incoming requests
type ClientHandler func(payload []byte) ([]byte, error)

// EventHandler handles incoming events
type EventHandler func(event string, payload []byte)

// NewClient creates a new IPC client with default configuration
func NewClient(serviceName string) *IPCClient {
	return NewClientWithConfig(DefaultClientConfig(serviceName))
}

// NewClientWithConfig creates a new IPC client with custom configuration
func NewClientWithConfig(config ClientConfig) *IPCClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &IPCClient{
		config:        config,
		pending:       make(map[uint64]chan *pb.IPCMessage),
		handlers:      make(map[string]ClientHandler),
		eventHandlers: make(map[string]EventHandler),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Connect connects to the IPC server
func (c *IPCClient) Connect() error {
	return c.ConnectWithRetry(c.config.MaxRetries)
}

// ConnectWithRetry connects with retry logic
func (c *IPCClient) ConnectWithRetry(maxRetries int) error {
	var lastErr error
	delay := c.config.ReconnectDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		select {
		case <-c.ctx.Done():
			return c.ctx.Err()
		default:
		}

		if attempt > 0 {
			time.Sleep(delay)
			// Exponential backoff
			delay = delay * 2
			if delay > c.config.MaxReconnectDelay {
				delay = c.config.MaxReconnectDelay
			}
		}

		err := c.connect()
		if err == nil {
			return nil
		}
		lastErr = err
		fmt.Printf("[ipc-client] Connection attempt %d failed: %v\n", attempt+1, err)
	}

	return fmt.Errorf("failed to connect after %d attempts: %w", maxRetries+1, lastErr)
}

func (c *IPCClient) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Create connection with timeout
	dialer := net.Dialer{Timeout: c.config.ConnectTimeout}
	conn, err := dialer.Dial("unix", c.config.SocketPath)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	c.conn = conn
	c.connected = true

	// Start receive loop
	go c.receiveLoop()

	// Register with broker
	if err := c.register(); err != nil {
		c.conn.Close()
		c.connected = false
		return fmt.Errorf("registration failed: %w", err)
	}

	// Start heartbeat
	go c.heartbeatLoop()

	return nil
}

// register registers the client with the broker
func (c *IPCClient) register() error {
	payload, _ := json.Marshal(map[string]string{
		"service": c.config.ServiceName,
		"type":    "client",
	})

	response, err := c.Call("broker", "register", payload)
	if err != nil {
		return err
	}

	if response.Error != "" {
		return fmt.Errorf("registration error: %s", response.Error)
	}

	fmt.Printf("[ipc-client] Registered as %s\n", c.config.ServiceName)
	return nil
}

// receiveLoop handles incoming messages
func (c *IPCClient) receiveLoop() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		c.mu.RLock()
		conn := c.conn
		connected := c.connected
		c.mu.RUnlock()

		if !connected || conn == nil {
			return
		}

		msg, err := c.readMessage(conn)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("[ipc-client] Read error: %v\n", err)
			}
			c.handleDisconnect()
			return
		}

		c.handleMessage(msg)
	}
}

// handleMessage processes an incoming message
func (c *IPCClient) handleMessage(msg *pb.IPCMessage) {
	switch msg.MsgType {
	case pb.MessageType_RESPONSE:
		// Check if this is a response to a pending request
		c.pendingMu.RLock()
		ch, ok := c.pending[msg.MsgId]
		c.pendingMu.RUnlock()

		if ok {
			select {
			case ch <- msg:
			default:
			}
		}

	case pb.MessageType_REQUEST:
		// Handle incoming request
		if msg.Method == "ping" {
			// Respond to heartbeat
			c.sendPong(msg)
			return
		}

		c.handlersMu.RLock()
		handler, ok := c.handlers[msg.Method]
		c.handlersMu.RUnlock()

		if ok {
			go c.handleRequest(msg, handler)
		}

	case pb.MessageType_EVENT:
		// Handle event
		c.eventMu.RLock()
		handler, ok := c.eventHandlers[msg.Method]
		c.eventMu.RUnlock()

		if ok {
			go handler(msg.Method, msg.Payload)
		}

		// Also check for wildcard handlers
		c.eventMu.RLock()
		wildcardHandler, ok := c.eventHandlers["*"]
		c.eventMu.RUnlock()

		if ok {
			go wildcardHandler(msg.Method, msg.Payload)
		}
	}
}

// handleRequest handles an incoming request
func (c *IPCClient) handleRequest(msg *pb.IPCMessage, handler ClientHandler) {
	result, err := handler(msg.Payload)

	response := &pb.IPCMessage{
		Version:   1,
		MsgType:   pb.MessageType_RESPONSE,
		MsgId:     msg.MsgId,
		Source:    c.config.ServiceName,
		Target:    msg.Source,
		Method:    msg.Method,
		Timestamp: uint64(time.Now().UnixMilli()),
	}

	if err != nil {
		response.Error = err.Error()
	} else {
		response.Payload = result
	}

	c.sendMessage(response)
}

// sendPong responds to a ping
func (c *IPCClient) sendPong(ping *pb.IPCMessage) {
	pong := &pb.IPCMessage{
		Version:   1,
		MsgType:   pb.MessageType_RESPONSE,
		MsgId:     ping.MsgId,
		Source:    c.config.ServiceName,
		Target:    ping.Source,
		Method:    "pong",
		Payload:   []byte("pong"),
		Timestamp: uint64(time.Now().UnixMilli()),
	}
	c.sendMessage(pong)
}

// heartbeatLoop sends periodic heartbeats
func (c *IPCClient) heartbeatLoop() {
	ticker := time.NewTicker(c.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			connected := c.connected
			c.mu.RUnlock()

			if !connected {
				return
			}

			// Send ping to init
			_, err := c.CallWithTimeout("init", "ping", []byte("ping"), 5*time.Second)
			if err != nil {
				fmt.Printf("[ipc-client] Heartbeat failed: %v\n", err)
			}
		}
	}
}

// handleDisconnect handles connection loss
func (c *IPCClient) handleDisconnect() {
	c.mu.Lock()
	c.connected = false
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()

	// Cancel all pending requests
	c.pendingMu.Lock()
	for id, ch := range c.pending {
		close(ch)
		delete(c.pending, id)
	}
	c.pendingMu.Unlock()

	// Attempt reconnection
	c.reconnect()
}

// reconnect attempts to reconnect
func (c *IPCClient) reconnect() {
	c.reconnectMu.Lock()
	if c.reconnecting {
		c.reconnectMu.Unlock()
		return
	}
	c.reconnecting = true
	c.reconnectMu.Unlock()

	defer func() {
		c.reconnectMu.Lock()
		c.reconnecting = false
		c.reconnectMu.Unlock()
	}()

	fmt.Println("[ipc-client] Attempting to reconnect...")
	
	delay := c.config.ReconnectDelay
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		time.Sleep(delay)

		err := c.connect()
		if err == nil {
			fmt.Println("[ipc-client] Reconnected successfully")
			return
		}

		fmt.Printf("[ipc-client] Reconnection failed: %v\n", err)
		
		// Exponential backoff
		delay = delay * 2
		if delay > c.config.MaxReconnectDelay {
			delay = c.config.MaxReconnectDelay
		}
	}
}

// Call makes an RPC call to a service
func (c *IPCClient) Call(target, method string, payload []byte) (*pb.IPCMessage, error) {
	return c.CallWithTimeout(target, method, payload, c.config.RequestTimeout)
}

// CallWithTimeout makes an RPC call with a custom timeout
func (c *IPCClient) CallWithTimeout(target, method string, payload []byte, timeout time.Duration) (*pb.IPCMessage, error) {
	c.mu.RLock()
	if !c.connected {
		c.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	c.mu.RUnlock()

	msgID := atomic.AddUint64(&c.msgCounter, 1)

	msg := &pb.IPCMessage{
		Version:   1,
		MsgType:   pb.MessageType_REQUEST,
		MsgId:     msgID,
		Source:    c.config.ServiceName,
		Target:    target,
		Method:    method,
		Payload:   payload,
		Timestamp: uint64(time.Now().UnixMilli()),
	}

	// Create response channel
	responseCh := make(chan *pb.IPCMessage, 1)
	c.pendingMu.Lock()
	c.pending[msgID] = responseCh
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, msgID)
		c.pendingMu.Unlock()
	}()

	// Send message
	if err := c.sendMessage(msg); err != nil {
		return nil, fmt.Errorf("send failed: %w", err)
	}

	// Wait for response
	select {
	case response := <-responseCh:
		if response == nil {
			return nil, fmt.Errorf("connection closed")
		}
		return response, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("request timeout")
	case <-c.ctx.Done():
		return nil, c.ctx.Err()
	}
}

// CallJSON makes an RPC call with JSON payload and response
func (c *IPCClient) CallJSON(target, method string, request interface{}, response interface{}) error {
	payload, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	msg, err := c.Call(target, method, payload)
	if err != nil {
		return err
	}

	if msg.Error != "" {
		return fmt.Errorf("service error: %s", msg.Error)
	}

	if response != nil {
		if err := json.Unmarshal(msg.Payload, response); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

// SendEvent sends an event
func (c *IPCClient) SendEvent(event string, payload []byte) error {
	msg := &pb.IPCMessage{
		Version:   1,
		MsgType:   pb.MessageType_EVENT,
		MsgId:     atomic.AddUint64(&c.msgCounter, 1),
		Source:    c.config.ServiceName,
		Target:    "*",
		Method:    event,
		Payload:   payload,
		Timestamp: uint64(time.Now().UnixMilli()),
	}
	return c.sendMessage(msg)
}

// RegisterHandler registers a handler for incoming requests
func (c *IPCClient) RegisterHandler(method string, handler ClientHandler) {
	c.handlersMu.Lock()
	defer c.handlersMu.Unlock()
	c.handlers[method] = handler
}

// RegisterEventHandler registers a handler for events
func (c *IPCClient) RegisterEventHandler(event string, handler EventHandler) {
	c.eventMu.Lock()
	defer c.eventMu.Unlock()
	c.eventHandlers[event] = handler
}

// Subscribe subscribes to events
func (c *IPCClient) Subscribe(events []string) error {
	payload := []byte(fmt.Sprintf(`{"events":["%s"]}`, 
		func() string {
			result := ""
			for i, e := range events {
				if i > 0 {
					result += `","`
				}
				result += e
			}
			return result
		}()))
	
	_, err := c.Call("broker", "subscribe", payload)
	return err
}

// sendMessage sends a message to the server
func (c *IPCClient) sendMessage(msg *pb.IPCMessage) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Write length prefix
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))
	if _, err := conn.Write(lenBuf); err != nil {
		return err
	}

	// Write message
	_, err = conn.Write(data)
	return err
}

// readMessage reads a message from the connection
func (c *IPCClient) readMessage(conn net.Conn) (*pb.IPCMessage, error) {
	// Read length prefix
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf)

	// Sanity check
	if length > 16*1024*1024 {
		return nil, fmt.Errorf("message too large: %d bytes", length)
	}

	// Read message
	msgBuf := make([]byte, length)
	if _, err := io.ReadFull(conn, msgBuf); err != nil {
		return nil, err
	}

	var msg pb.IPCMessage
	if err := json.Unmarshal(msgBuf, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

// IsConnected returns whether the client is connected
func (c *IPCClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Close closes the client connection
func (c *IPCClient) Close() error {
	c.cancel()

	c.mu.Lock()
	defer c.mu.Unlock()

	c.connected = false
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// Convenience functions for common service calls

// CallPkgMgr calls the package manager service
func (c *IPCClient) CallPkgMgr(action string, packages []string, options map[string]string) (map[string]interface{}, error) {
	request := map[string]interface{}{
		"action":   action,
		"packages": packages,
		"options":  options,
	}
	var response map[string]interface{}
	err := c.CallJSON("pkgmgr", "package", request, &response)
	return response, err
}

// CallBuilder calls the builder service
func (c *IPCClient) CallBuilder(packageName, sourcePath, outputPath string, env map[string]string) (map[string]interface{}, error) {
	request := map[string]interface{}{
		"package_name": packageName,
		"source_path":  sourcePath,
		"output_path":  outputPath,
		"env":          env,
		"build_args":   []string{},
	}
	var response map[string]interface{}
	err := c.CallJSON("builder", "build", request, &response)
	return response, err
}

// CallResolver calls the resolver service
func (c *IPCClient) CallResolver(packages []string, includeOptional bool) (map[string]interface{}, error) {
	request := map[string]interface{}{
		"packages":         packages,
		"include_optional": includeOptional,
	}
	var response map[string]interface{}
	err := c.CallJSON("resolver", "resolve", request, &response)
	return response, err
}

// CallCache calls the cache service
func (c *IPCClient) CallCache(action, key string, value *string, ttl int64) (map[string]interface{}, error) {
	request := map[string]interface{}{
		"action": action,
		"key":    key,
		"ttl":    ttl,
	}
	if value != nil {
		request["value"] = *value
	}
	var response map[string]interface{}
	err := c.CallJSON("cache", "cache", request, &response)
	return response, err
}
