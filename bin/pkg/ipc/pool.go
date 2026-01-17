package ipc

import (
	"net"
	"sync"
	"time"
)

// ConnectionPool manages a pool of client connections
type ConnectionPool struct {
	maxSize     int
	idleTimeout time.Duration
	connections map[string]*pooledConnection
	mu          sync.RWMutex
}

type pooledConnection struct {
	conn       net.Conn
	service    string
	lastUsed   time.Time
	inUse      bool
	healthy    bool
	created    time.Time
	requestCnt int64
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxSize int, idleTimeout time.Duration) *ConnectionPool {
	pool := &ConnectionPool{
		maxSize:     maxSize,
		idleTimeout: idleTimeout,
		connections: make(map[string]*pooledConnection),
	}
	go pool.cleanupLoop()
	return pool
}

// Add adds a connection to the pool
func (p *ConnectionPool) Add(id string, conn net.Conn, service string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.connections[id] = &pooledConnection{
		conn:     conn,
		service:  service,
		lastUsed: time.Now(),
		inUse:    true,
		healthy:  true,
		created:  time.Now(),
	}
}

// Get retrieves a connection by ID
func (p *ConnectionPool) Get(id string) (*pooledConnection, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	conn, ok := p.connections[id]
	if ok {
		conn.lastUsed = time.Now()
	}
	return conn, ok
}

// GetByService retrieves connections by service name
func (p *ConnectionPool) GetByService(service string) []*pooledConnection {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var conns []*pooledConnection
	for _, conn := range p.connections {
		if conn.service == service && conn.healthy {
			conns = append(conns, conn)
		}
	}
	return conns
}

// Remove removes a connection from the pool
func (p *ConnectionPool) Remove(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, ok := p.connections[id]; ok {
		conn.conn.Close()
		delete(p.connections, id)
	}
}

// MarkUnhealthy marks a connection as unhealthy
func (p *ConnectionPool) MarkUnhealthy(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, ok := p.connections[id]; ok {
		conn.healthy = false
	}
}

// MarkHealthy marks a connection as healthy
func (p *ConnectionPool) MarkHealthy(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, ok := p.connections[id]; ok {
		conn.healthy = true
	}
}

// IncrementRequestCount increments the request counter for a connection
func (p *ConnectionPool) IncrementRequestCount(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, ok := p.connections[id]; ok {
		conn.requestCnt++
	}
}

// Size returns the current pool size
func (p *ConnectionPool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.connections)
}

// HealthyCount returns the number of healthy connections
func (p *ConnectionPool) HealthyCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, conn := range p.connections {
		if conn.healthy {
			count++
		}
	}
	return count
}

// Stats returns pool statistics
func (p *ConnectionPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := PoolStats{
		TotalConnections:   len(p.connections),
		HealthyConnections: 0,
		ServiceCounts:      make(map[string]int),
	}

	for _, conn := range p.connections {
		if conn.healthy {
			stats.HealthyConnections++
		}
		stats.ServiceCounts[conn.service]++
		stats.TotalRequests += conn.requestCnt
	}

	return stats
}

// PoolStats contains connection pool statistics
type PoolStats struct {
	TotalConnections   int
	HealthyConnections int
	ServiceCounts      map[string]int
	TotalRequests      int64
}

// cleanupLoop periodically removes idle connections
func (p *ConnectionPool) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		p.cleanup()
	}
}

// cleanup removes idle and unhealthy connections
func (p *ConnectionPool) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for id, conn := range p.connections {
		// Remove idle connections
		if !conn.inUse && now.Sub(conn.lastUsed) > p.idleTimeout {
			conn.conn.Close()
			delete(p.connections, id)
			continue
		}
		// Remove unhealthy connections that have been unhealthy for too long
		if !conn.healthy && now.Sub(conn.lastUsed) > p.idleTimeout {
			conn.conn.Close()
			delete(p.connections, id)
		}
	}
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for id, conn := range p.connections {
		conn.conn.Close()
		delete(p.connections, id)
	}
}

// All returns all connections (for iteration)
func (p *ConnectionPool) All() map[string]*pooledConnection {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return a copy to avoid race conditions
	copy := make(map[string]*pooledConnection)
	for k, v := range p.connections {
		copy[k] = v
	}
	return copy
}
