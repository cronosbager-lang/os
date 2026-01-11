package ipc

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"

	pb "mixos.dev/init/pkg/ipc/proto"
)

type Server struct {
	socketPath string
	listener   net.Listener
	clients    map[string]*Client
	handlers   map[string]Handler
	mu         sync.RWMutex
	done       chan struct{}
}

type Client struct {
	ID       string
	Conn     net.Conn
	Service  string
}

type Handler func(msg *pb.IPCMessage) (*pb.IPCMessage, error)

func NewServer(socketPath string) (*Server, error) {
	// Ensure directory exists
	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket
	os.Remove(socketPath)

	return &Server{
		socketPath: socketPath,
		clients:    make(map[string]*Client),
		handlers:   make(map[string]Handler),
		done:       make(chan struct{}),
	}, nil
}

func (s *Server) RegisterHandler(method string, handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = handler
}

func (s *Server) Listen() error {
	var err error
	s.listener, err = net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Set socket permissions
	os.Chmod(s.socketPath, 0660)

	for {
		select {
		case <-s.done:
			return nil
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				continue
			}
			go s.handleConnection(conn)
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	client := &Client{
		ID:   fmt.Sprintf("%p", conn),
		Conn: conn,
	}

	s.mu.Lock()
	s.clients[client.ID] = client
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, client.ID)
		s.mu.Unlock()
	}()

	for {
		msg, err := s.readMessage(conn)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("[ipc] Read error: %v\n", err)
			}
			return
		}

		// Handle registration
		if msg.Method == "register" {
			client.Service = msg.Source
			response := &pb.IPCMessage{
				Version: 1,
				MsgType: pb.MessageType_RESPONSE,
				MsgId:   msg.MsgId,
				Source:  "init",
				Target:  msg.Source,
				Method:  "register",
				Payload: []byte("ok"),
			}
			s.writeMessage(conn, response)
			continue
		}

		// Route message
		response := s.routeMessage(msg)
		if err := s.writeMessage(conn, response); err != nil {
			fmt.Printf("[ipc] Write error: %v\n", err)
			return
		}
	}
}

func (s *Server) routeMessage(msg *pb.IPCMessage) *pb.IPCMessage {
	// Check if target is a registered service
	if msg.Target != "" && msg.Target != "init" {
		s.mu.RLock()
		for _, client := range s.clients {
			if client.Service == msg.Target {
				s.mu.RUnlock()
				// Forward to target service
				return s.forwardMessage(client, msg)
			}
		}
		s.mu.RUnlock()

		return &pb.IPCMessage{
			Version: 1,
			MsgType: pb.MessageType_RESPONSE,
			MsgId:   msg.MsgId,
			Source:  "init",
			Target:  msg.Source,
			Method:  msg.Method,
			Error:   fmt.Sprintf("service not found: %s", msg.Target),
		}
	}

	// Handle locally
	s.mu.RLock()
	handler, ok := s.handlers[msg.Method]
	s.mu.RUnlock()

	if !ok {
		return &pb.IPCMessage{
			Version: 1,
			MsgType: pb.MessageType_RESPONSE,
			MsgId:   msg.MsgId,
			Source:  "init",
			Target:  msg.Source,
			Method:  msg.Method,
			Error:   fmt.Sprintf("unknown method: %s", msg.Method),
		}
	}

	response, err := handler(msg)
	if err != nil {
		return &pb.IPCMessage{
			Version: 1,
			MsgType: pb.MessageType_RESPONSE,
			MsgId:   msg.MsgId,
			Source:  "init",
			Target:  msg.Source,
			Method:  msg.Method,
			Error:   err.Error(),
		}
	}

	return response
}

func (s *Server) forwardMessage(client *Client, msg *pb.IPCMessage) *pb.IPCMessage {
	if err := s.writeMessage(client.Conn, msg); err != nil {
		return &pb.IPCMessage{
			Version: 1,
			MsgType: pb.MessageType_RESPONSE,
			MsgId:   msg.MsgId,
			Source:  "init",
			Target:  msg.Source,
			Method:  msg.Method,
			Error:   fmt.Sprintf("forward failed: %v", err),
		}
	}

	response, err := s.readMessage(client.Conn)
	if err != nil {
		return &pb.IPCMessage{
			Version: 1,
			MsgType: pb.MessageType_RESPONSE,
			MsgId:   msg.MsgId,
			Source:  "init",
			Target:  msg.Source,
			Method:  msg.Method,
			Error:   fmt.Sprintf("response failed: %v", err),
		}
	}

	return response
}

func (s *Server) readMessage(conn net.Conn) (*pb.IPCMessage, error) {
	// Read length prefix (4 bytes)
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf)

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

func (s *Server) Broadcast(msg *pb.IPCMessage) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		s.writeMessage(client.Conn, msg)
	}
}

func (s *Server) Close() error {
	close(s.done)
	if s.listener != nil {
		s.listener.Close()
	}
	os.Remove(s.socketPath)
	return nil
}
