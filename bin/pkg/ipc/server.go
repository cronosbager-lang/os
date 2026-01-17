package ipc

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	pb "mixos.dev/init/pkg/ipc/proto"
)

// ServerConfig contains IPC server configuration
type ServerConfig struct {
	SocketPath       string
	MaxConnections   int
	IdleTimeout      time.Duration
	HeartbeatInterval time.Duration
	HeartbeatTimeout  time.Duration
	MessageTimeout   time.Duration
	MaxRetries       int
	MaxQueueSize     int
}

// DefaultServerConfig returns default server configuration
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		SocketPath:        "/run/mixos/ipc.sock",
		MaxConnections:    100,
		IdleTimeout:       5 * time.Minute,
		HeartbeatInterval: 30 * time.Second,
		HeartbeatTimeout:  10 * time.Second,
		MessageTimeout:    30 * time.Second,
		MaxRetries:        3,
		MaxQueueSize:      1000,
	}
}

type Server struct {
	socketPath string
	listener   net.Listener
	clients    map[string]*Client
	handlers   map[string]Handler
	mu         sync.RWMutex
	done       chan struct{}
	
	// Enhanced features
	config     ServerConfig
	pool       *ConnectionPool
	heartbeat  *HeartbeatManager
	tracker    *MessageTracker
	queue      *MessageQueue
	ctx        context.Context
	cancel     context.CancelFunc
	
	// Metrics
	metrics    *ServerMetrics
}

// ServerMetrics tracks server statistics
type ServerMetrics struct {
	MessagesReceived  int64
	MessagesSent      int64
	MessagesForwarded int64
	Errors            int64
	StartTime         time.Time
	mu                sync.RWMutex
}

func (m *ServerMetrics) IncrReceived() {
	m.mu.Lock()
	m.MessagesReceived++
	m.mu.Unlock()
}

func (m *ServerMetrics) IncrSent() {
	m.mu.Lock()
	m.MessagesSent++
	m.mu.Unlock()
}

func (m *ServerMetrics) IncrForwarded() {
	m.mu.Lock()
	m.MessagesForwarded++
	m.mu.Unlock()
}

func (m *ServerMetrics) IncrErrors() {
	m.mu.Lock()
	m.Errors++
	m.mu.Unlock()
}

func (m *ServerMetrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return map[string]interface{}{
		"messages_received":  m.MessagesReceived,
		"messages_sent":      m.MessagesSent,
		"messages_forwarded": m.MessagesForwarded,
		"errors":             m.Errors,
		"uptime_seconds":     time.Since(m.StartTime).Seconds(),
	}
}

type Client struct {
	ID       string
	Conn     net.Conn
	Service  string
	ConnectedAt time.Time
	LastActivity time.Time
}

type Handler func(msg *pb.IPCMessage) (*pb.IPCMessage, error)

// NewServer creates a new IPC server with default configuration
func NewServer(socketPath string) (*Server, error) {
	config := DefaultServerConfig()
	config.SocketPath = socketPath
	return NewServerWithConfig(config)
}

// NewServerWithConfig creates a new IPC server with custom configuration
func NewServerWithConfig(config ServerConfig) (*Server, error) {
	// Ensure directory exists
	dir := filepath.Dir(config.SocketPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket
	os.Remove(config.SocketPath)

	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		socketPath: config.SocketPath,
		clients:    make(map[string]*Client),
		handlers:   make(map[string]Handler),
		done:       make(chan struct{}),
		config:     config,
		pool:       NewConnectionPool(config.MaxConnections, config.IdleTimeout),
		tracker:    NewMessageTracker(config.MessageTimeout, config.MaxRetries),
		queue:      NewMessageQueue(config.MaxQueueSize),
		ctx:        ctx,
		cancel:     cancel,
		metrics:    &ServerMetrics{StartTime: time.Now()},
	}

	// Initialize heartbeat manager (needs server reference)
	server.heartbeat = NewHeartbeatManager(server, config.HeartbeatInterval, config.HeartbeatTimeout)

	// Register built-in handlers
	server.registerBuiltinHandlers()

	return server, nil
}

func (s *Server) RegisterHandler(method string, handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = handler
}

// registerBuiltinHandlers registers built-in IPC handlers
func (s *Server) registerBuiltinHandlers() {
	// Ping handler for heartbeat
	s.handlers["ping"] = func(msg *pb.IPCMessage) (*pb.IPCMessage, error) {
		return &pb.IPCMessage{
			Version:   1,
			MsgType:   pb.MessageType_RESPONSE,
			MsgId:     msg.MsgId,
			Source:    "init",
			Target:    msg.Source,
			Method:    "pong",
			Payload:   []byte("pong"),
			Timestamp: uint64(time.Now().UnixMilli()),
		}, nil
	}

	// Stats handler
	s.handlers["stats"] = func(msg *pb.IPCMessage) (*pb.IPCMessage, error) {
		stats := s.GetStats()
		data, _ := json.Marshal(stats)
		return &pb.IPCMessage{
			Version:   1,
			MsgType:   pb.MessageType_RESPONSE,
			MsgId:     msg.MsgId,
			Source:    "init",
			Target:    msg.Source,
			Method:    "stats",
			Payload:   data,
			Timestamp: uint64(time.Now().UnixMilli()),
		}, nil
	}

	// List services handler
	s.handlers["list_services"] = func(msg *pb.IPCMessage) (*pb.IPCMessage, error) {
		services := s.ListServices()
		data, _ := json.Marshal(services)
		return &pb.IPCMessage{
			Version:   1,
			MsgType:   pb.MessageType_RESPONSE,
			MsgId:     msg.MsgId,
			Source:    "init",
			Target:    msg.Source,
			Method:    "list_services",
			Payload:   data,
			Timestamp: uint64(time.Now().UnixMilli()),
		}, nil
	}
}

// Listen starts the IPC server
func (s *Server) Listen() error {
	var err error
	s.listener, err = net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Set socket permissions
	os.Chmod(s.socketPath, 0660)

	// Start heartbeat manager
	s.heartbeat.Start()

	// Start retry loop
	go s.retryLoop()

	for {
		select {
		case <-s.done:
			return nil
		case <-s.ctx.Done():
			return nil
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.done:
					return nil
				default:
					continue
				}
			}
			
			// Check max connections
			if s.pool.Size() >= s.config.MaxConnections {
				conn.Close()
				continue
			}
			
			go s.handleConnection(conn)
		}
	}
}

// retryLoop handles message retries
func (s *Server) retryLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			msgs := s.tracker.GetRetryMessages()
			for _, msg := range msgs {
				s.routeMessage(msg)
			}
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	now := time.Now()
	client := &Client{
		ID:          fmt.Sprintf("%p", conn),
		Conn:        conn,
		ConnectedAt: now,
		LastActivity: now,
	}

	s.mu.Lock()
	s.clients[client.ID] = client
	s.mu.Unlock()

	// Add to connection pool
	s.pool.Add(client.ID, conn, "")

	defer func() {
		s.mu.Lock()
		delete(s.clients, client.ID)
		s.mu.Unlock()
		s.pool.Remove(client.ID)
		s.heartbeat.RemoveClient(client.ID)
		conn.Close()
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		msg, err := s.readMessage(conn)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("[ipc] Read error: %v\n", err)
				s.metrics.IncrErrors()
			}
			return
		}

		s.metrics.IncrReceived()
		client.LastActivity = time.Now()

		// Handle pong response (heartbeat)
		if msg.Method == "pong" {
			s.heartbeat.RecordPong(client.ID)
			continue
		}

		// Handle registration
		if msg.Method == "register" {
			client.Service = msg.Source
			s.pool.Remove(client.ID)
			s.pool.Add(client.ID, conn, msg.Source)
			
			response := &pb.IPCMessage{
				Version:   1,
				MsgType:   pb.MessageType_RESPONSE,
				MsgId:     msg.MsgId,
				Source:    "init",
				Target:    msg.Source,
				Method:    "register",
				Payload:   []byte(`{"status":"ok"}`),
				Timestamp: uint64(time.Now().UnixMilli()),
			}
			s.writeMessage(conn, response)
			s.metrics.IncrSent()
			
			// Deliver any queued messages
			s.deliverQueuedMessages(msg.Source)
			continue
		}

		// Handle response to tracked message
		if msg.MsgType == pb.MessageType_RESPONSE {
			if s.tracker.Acknowledge(msg.MsgId, msg) {
				continue
			}
		}

		// Route message
		response := s.routeMessage(msg)
		if err := s.writeMessage(conn, response); err != nil {
			fmt.Printf("[ipc] Write error: %v\n", err)
			s.metrics.IncrErrors()
			return
		}
		s.metrics.IncrSent()
	}
}

// deliverQueuedMessages delivers queued messages to a newly connected service
func (s *Server) deliverQueuedMessages(service string) {
	msgs := s.queue.Dequeue(service)
	for _, msg := range msgs {
		s.routeMessage(msg)
	}
}

func (s *Server) routeMessage(msg *pb.IPCMessage) *pb.IPCMessage {
	// Check if target is a registered service
	if msg.Target != "" && msg.Target != "init" && msg.Target != "*" {
		s.mu.RLock()
		var targetClient *Client
		for _, client := range s.clients {
			if client.Service == msg.Target {
				targetClient = client
				break
			}
		}
		s.mu.RUnlock()

		if targetClient != nil {
			// Forward to target service
			s.metrics.IncrForwarded()
			return s.forwardMessage(targetClient, msg)
		}

		// Service not found - queue message for later delivery
		if s.queue.Enqueue(msg.Target, msg) {
			return &pb.IPCMessage{
				Version:   1,
				MsgType:   pb.MessageType_RESPONSE,
				MsgId:     msg.MsgId,
				Source:    "init",
				Target:    msg.Source,
				Method:    msg.Method,
				Payload:   []byte(`{"status":"queued"}`),
				Timestamp: uint64(time.Now().UnixMilli()),
			}
		}

		return &pb.IPCMessage{
			Version:   1,
			MsgType:   pb.MessageType_RESPONSE,
			MsgId:     msg.MsgId,
			Source:    "init",
			Target:    msg.Source,
			Method:    msg.Method,
			Error:     fmt.Sprintf("service not found: %s", msg.Target),
			Timestamp: uint64(time.Now().UnixMilli()),
		}
	}

	// Handle broadcast
	if msg.Target == "*" {
		s.Broadcast(msg)
		return &pb.IPCMessage{
			Version:   1,
			MsgType:   pb.MessageType_RESPONSE,
			MsgId:     msg.MsgId,
			Source:    "init",
			Target:    msg.Source,
			Method:    msg.Method,
			Payload:   []byte(`{"status":"broadcast"}`),
			Timestamp: uint64(time.Now().UnixMilli()),
		}
	}

	// Handle locally
	s.mu.RLock()
	handler, ok := s.handlers[msg.Method]
	s.mu.RUnlock()

	if !ok {
		return &pb.IPCMessage{
			Version:   1,
			MsgType:   pb.MessageType_RESPONSE,
			MsgId:     msg.MsgId,
			Source:    "init",
			Target:    msg.Source,
			Method:    msg.Method,
			Error:     fmt.Sprintf("unknown method: %s", msg.Method),
			Timestamp: uint64(time.Now().UnixMilli()),
		}
	}

	response, err := handler(msg)
	if err != nil {
		return &pb.IPCMessage{
			Version:   1,
			MsgType:   pb.MessageType_RESPONSE,
			MsgId:     msg.MsgId,
			Source:    "init",
			Target:    msg.Source,
			Method:    msg.Method,
			Error:     err.Error(),
			Timestamp: uint64(time.Now().UnixMilli()),
		}
	}

	return response
}

func (s *Server) forwardMessage(client *Client, msg *pb.IPCMessage) *pb.IPCMessage {
	if err := s.writeMessage(client.Conn, msg); err != nil {
		s.pool.MarkUnhealthy(client.ID)
		return &pb.IPCMessage{
			Version:   1,
			MsgType:   pb.MessageType_RESPONSE,
			MsgId:     msg.MsgId,
			Source:    "init",
			Target:    msg.Source,
			Method:    msg.Method,
			Error:     fmt.Sprintf("forward failed: %v", err),
			Timestamp: uint64(time.Now().UnixMilli()),
		}
	}

	// Set read deadline for response
	client.Conn.SetReadDeadline(time.Now().Add(s.config.MessageTimeout))
	defer client.Conn.SetReadDeadline(time.Time{})

	response, err := s.readMessage(client.Conn)
	if err != nil {
		s.pool.MarkUnhealthy(client.ID)
		return &pb.IPCMessage{
			Version:   1,
			MsgType:   pb.MessageType_RESPONSE,
			MsgId:     msg.MsgId,
			Source:    "init",
			Target:    msg.Source,
			Method:    msg.Method,
			Error:     fmt.Sprintf("response failed: %v", err),
			Timestamp: uint64(time.Now().UnixMilli()),
		}
	}

	s.pool.IncrementRequestCount(client.ID)
	return response
}

func (s *Server) readMessage(conn net.Conn) (*pb.IPCMessage, error) {
	// Read length prefix (4 bytes)
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf)

	// Sanity check on message size (max 16MB)
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

func (s *Server) writeMessage(conn net.Conn, msg *pb.IPCMessage) error {
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

// Broadcast sends a message to all connected clients
func (s *Server) Broadcast(msg *pb.IPCMessage) {
	s.mu.RLock()
	clients := make([]*Client, 0, len(s.clients))
	for _, client := range s.clients {
		if client.Service != "" { // Only broadcast to registered services
			clients = append(clients, client)
		}
	}
	s.mu.RUnlock()

	for _, client := range clients {
		if err := s.writeMessage(client.Conn, msg); err != nil {
			s.pool.MarkUnhealthy(client.ID)
		}
	}
}

// BroadcastEvent sends an event to all subscribers
func (s *Server) BroadcastEvent(event string, payload []byte) {
	msg := &pb.IPCMessage{
		Version:   1,
		MsgType:   pb.MessageType_EVENT,
		MsgId:     s.tracker.NextMessageID(),
		Source:    "init",
		Target:    "*",
		Method:    event,
		Payload:   payload,
		Timestamp: uint64(time.Now().UnixMilli()),
	}
	s.Broadcast(msg)
}

// GetStats returns server statistics
func (s *Server) GetStats() map[string]interface{} {
	stats := s.metrics.GetStats()
	stats["pool"] = s.pool.Stats()
	stats["heartbeat"] = s.heartbeat.GetStats()
	stats["tracker"] = s.tracker.Stats()
	stats["queue_size"] = s.queue.TotalSize()
	return stats
}

// ListServices returns a list of connected services
func (s *Server) ListServices() []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	services := make([]map[string]interface{}, 0, len(s.clients))
	for _, client := range s.clients {
		if client.Service != "" {
			services = append(services, map[string]interface{}{
				"name":         client.Service,
				"connected_at": client.ConnectedAt,
				"last_activity": client.LastActivity,
				"healthy":      s.heartbeat.IsHealthy(client.ID),
			})
		}
	}
	return services
}

// GetClient returns a client by service name
func (s *Server) GetClient(service string) (*Client, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		if client.Service == service {
			return client, true
		}
	}
	return nil, false
}

// IsServiceConnected checks if a service is connected
func (s *Server) IsServiceConnected(service string) bool {
	_, ok := s.GetClient(service)
	return ok
}

// GracefulShutdown performs a graceful shutdown
func (s *Server) GracefulShutdown(timeout time.Duration) error {
	// Broadcast shutdown event
	s.BroadcastEvent("system.shutdown", []byte(`{"reason":"shutdown"}`))

	// Wait for clients to disconnect or timeout
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		s.mu.RLock()
		count := len(s.clients)
		s.mu.RUnlock()
		if count == 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return s.Close()
}

// Close closes the server
func (s *Server) Close() error {
	// Signal shutdown
	s.cancel()
	close(s.done)

	// Stop heartbeat
	s.heartbeat.Stop()

	// Close all connections
	s.pool.Close()

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Remove socket file
	os.Remove(s.socketPath)

	return nil
}
