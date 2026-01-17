package ipc

import (
	"sync"
	"time"

	pb "mixos.dev/init/pkg/ipc/proto"
)

// HeartbeatManager manages heartbeat for all connections
type HeartbeatManager struct {
	server       *Server
	interval     time.Duration
	timeout      time.Duration
	lastPing     map[string]time.Time
	lastPong     map[string]time.Time
	mu           sync.RWMutex
	done         chan struct{}
	missedCounts map[string]int
	maxMissed    int
}

// NewHeartbeatManager creates a new heartbeat manager
func NewHeartbeatManager(server *Server, interval, timeout time.Duration) *HeartbeatManager {
	return &HeartbeatManager{
		server:       server,
		interval:     interval,
		timeout:      timeout,
		lastPing:     make(map[string]time.Time),
		lastPong:     make(map[string]time.Time),
		missedCounts: make(map[string]int),
		maxMissed:    3,
		done:         make(chan struct{}),
	}
}

// Start begins the heartbeat loop
func (h *HeartbeatManager) Start() {
	go h.heartbeatLoop()
}

// Stop stops the heartbeat manager
func (h *HeartbeatManager) Stop() {
	close(h.done)
}

// heartbeatLoop sends periodic heartbeats to all clients
func (h *HeartbeatManager) heartbeatLoop() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-h.done:
			return
		case <-ticker.C:
			h.sendHeartbeats()
			h.checkTimeouts()
		}
	}
}

// sendHeartbeats sends ping to all connected clients
func (h *HeartbeatManager) sendHeartbeats() {
	h.server.mu.RLock()
	clients := make([]*Client, 0, len(h.server.clients))
	for _, client := range h.server.clients {
		clients = append(clients, client)
	}
	h.server.mu.RUnlock()

	now := time.Now()
	pingMsg := &pb.IPCMessage{
		Version:   1,
		MsgType:   pb.MessageType_REQUEST,
		MsgId:     uint64(now.UnixNano()),
		Source:    "init",
		Target:    "",
		Method:    "ping",
		Payload:   []byte("ping"),
		Timestamp: uint64(now.UnixMilli()),
	}

	for _, client := range clients {
		h.mu.Lock()
		h.lastPing[client.ID] = now
		h.mu.Unlock()

		pingMsg.Target = client.Service
		if err := h.server.writeMessage(client.Conn, pingMsg); err != nil {
			h.incrementMissed(client.ID)
		}
	}
}

// checkTimeouts checks for clients that haven't responded
func (h *HeartbeatManager) checkTimeouts() {
	h.mu.RLock()
	defer h.mu.RUnlock()

	now := time.Now()
	for clientID, lastPing := range h.lastPing {
		lastPong, hasPong := h.lastPong[clientID]

		// If we've sent a ping but haven't received a pong within timeout
		if !hasPong || lastPong.Before(lastPing) {
			if now.Sub(lastPing) > h.timeout {
				h.server.pool.MarkUnhealthy(clientID)
			}
		}
	}
}

// RecordPong records a pong response from a client
func (h *HeartbeatManager) RecordPong(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.lastPong[clientID] = time.Now()
	h.missedCounts[clientID] = 0
	h.server.pool.MarkHealthy(clientID)
}

// incrementMissed increments the missed heartbeat count
func (h *HeartbeatManager) incrementMissed(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.missedCounts[clientID]++
	if h.missedCounts[clientID] >= h.maxMissed {
		h.server.pool.MarkUnhealthy(clientID)
	}
}

// RemoveClient removes a client from heartbeat tracking
func (h *HeartbeatManager) RemoveClient(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.lastPing, clientID)
	delete(h.lastPong, clientID)
	delete(h.missedCounts, clientID)
}

// IsHealthy checks if a client is responding to heartbeats
func (h *HeartbeatManager) IsHealthy(clientID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	missed, ok := h.missedCounts[clientID]
	if !ok {
		return true // New client, assume healthy
	}
	return missed < h.maxMissed
}

// GetStats returns heartbeat statistics
func (h *HeartbeatManager) GetStats() HeartbeatStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := HeartbeatStats{
		TotalClients:   len(h.lastPing),
		HealthyClients: 0,
		ClientStats:    make(map[string]ClientHeartbeatStats),
	}

	for clientID := range h.lastPing {
		clientStats := ClientHeartbeatStats{
			LastPing:    h.lastPing[clientID],
			MissedCount: h.missedCounts[clientID],
		}
		if pong, ok := h.lastPong[clientID]; ok {
			clientStats.LastPong = pong
		}
		if h.missedCounts[clientID] < h.maxMissed {
			stats.HealthyClients++
		}
		stats.ClientStats[clientID] = clientStats
	}

	return stats
}

// HeartbeatStats contains heartbeat statistics
type HeartbeatStats struct {
	TotalClients   int
	HealthyClients int
	ClientStats    map[string]ClientHeartbeatStats
}

// ClientHeartbeatStats contains per-client heartbeat stats
type ClientHeartbeatStats struct {
	LastPing    time.Time
	LastPong    time.Time
	MissedCount int
}
