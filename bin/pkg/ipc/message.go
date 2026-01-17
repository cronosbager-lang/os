package ipc

import (
	"sync"
	"sync/atomic"
	"time"

	pb "mixos.dev/init/pkg/ipc/proto"
)

// MessageTracker tracks pending messages and handles acknowledgments
type MessageTracker struct {
	pending      map[uint64]*pendingMessage
	mu           sync.RWMutex
	timeout      time.Duration
	retryCount   int
	retryBackoff time.Duration
	msgCounter   uint64
}

type pendingMessage struct {
	msg        *pb.IPCMessage
	sentAt     time.Time
	retries    int
	responseCh chan *pb.IPCMessage
	done       bool
}

// NewMessageTracker creates a new message tracker
func NewMessageTracker(timeout time.Duration, retryCount int) *MessageTracker {
	mt := &MessageTracker{
		pending:      make(map[uint64]*pendingMessage),
		timeout:      timeout,
		retryCount:   retryCount,
		retryBackoff: 100 * time.Millisecond,
	}
	go mt.timeoutLoop()
	return mt
}

// NextMessageID generates a unique message ID
func (mt *MessageTracker) NextMessageID() uint64 {
	return atomic.AddUint64(&mt.msgCounter, 1)
}

// Track starts tracking a message for acknowledgment
func (mt *MessageTracker) Track(msg *pb.IPCMessage) <-chan *pb.IPCMessage {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	responseCh := make(chan *pb.IPCMessage, 1)
	mt.pending[msg.MsgId] = &pendingMessage{
		msg:        msg,
		sentAt:     time.Now(),
		retries:    0,
		responseCh: responseCh,
	}
	return responseCh
}

// Acknowledge marks a message as acknowledged
func (mt *MessageTracker) Acknowledge(msgID uint64, response *pb.IPCMessage) bool {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if pm, ok := mt.pending[msgID]; ok && !pm.done {
		pm.done = true
		pm.responseCh <- response
		close(pm.responseCh)
		delete(mt.pending, msgID)
		return true
	}
	return false
}

// Cancel cancels tracking for a message
func (mt *MessageTracker) Cancel(msgID uint64) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if pm, ok := mt.pending[msgID]; ok {
		if !pm.done {
			close(pm.responseCh)
		}
		delete(mt.pending, msgID)
	}
}

// GetPending returns a pending message for retry
func (mt *MessageTracker) GetPending(msgID uint64) (*pb.IPCMessage, bool) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	if pm, ok := mt.pending[msgID]; ok {
		return pm.msg, true
	}
	return nil, false
}

// timeoutLoop checks for timed out messages
func (mt *MessageTracker) timeoutLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		mt.checkTimeouts()
	}
}

// checkTimeouts handles message timeouts
func (mt *MessageTracker) checkTimeouts() {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	now := time.Now()
	for msgID, pm := range mt.pending {
		if pm.done {
			continue
		}

		if now.Sub(pm.sentAt) > mt.timeout {
			if pm.retries < mt.retryCount {
				// Mark for retry
				pm.retries++
				pm.sentAt = now
			} else {
				// Max retries exceeded, timeout
				pm.done = true
				close(pm.responseCh)
				delete(mt.pending, msgID)
			}
		}
	}
}

// GetRetryMessages returns messages that need to be retried
func (mt *MessageTracker) GetRetryMessages() []*pb.IPCMessage {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	var msgs []*pb.IPCMessage
	now := time.Now()
	for _, pm := range mt.pending {
		if !pm.done && pm.retries > 0 {
			// Calculate backoff
			backoff := mt.retryBackoff * time.Duration(1<<uint(pm.retries-1))
			if now.Sub(pm.sentAt) > backoff {
				msgs = append(msgs, pm.msg)
			}
		}
	}
	return msgs
}

// Stats returns message tracker statistics
func (mt *MessageTracker) Stats() MessageTrackerStats {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	stats := MessageTrackerStats{
		PendingCount: len(mt.pending),
	}

	for _, pm := range mt.pending {
		if pm.retries > 0 {
			stats.RetryCount++
		}
	}

	return stats
}

// MessageTrackerStats contains message tracker statistics
type MessageTrackerStats struct {
	PendingCount int
	RetryCount   int
}

// MessageQueue provides a queue for messages when services are offline
type MessageQueue struct {
	queues   map[string][]*pb.IPCMessage
	maxSize  int
	mu       sync.RWMutex
}

// NewMessageQueue creates a new message queue
func NewMessageQueue(maxSize int) *MessageQueue {
	return &MessageQueue{
		queues:  make(map[string][]*pb.IPCMessage),
		maxSize: maxSize,
	}
}

// Enqueue adds a message to the queue for a service
func (mq *MessageQueue) Enqueue(service string, msg *pb.IPCMessage) bool {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	queue := mq.queues[service]
	if len(queue) >= mq.maxSize {
		return false // Queue full
	}

	mq.queues[service] = append(queue, msg)
	return true
}

// Dequeue removes and returns all messages for a service
func (mq *MessageQueue) Dequeue(service string) []*pb.IPCMessage {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	msgs := mq.queues[service]
	delete(mq.queues, service)
	return msgs
}

// Peek returns messages without removing them
func (mq *MessageQueue) Peek(service string) []*pb.IPCMessage {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	return mq.queues[service]
}

// Size returns the queue size for a service
func (mq *MessageQueue) Size(service string) int {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	return len(mq.queues[service])
}

// TotalSize returns the total number of queued messages
func (mq *MessageQueue) TotalSize() int {
	mq.mu.RLock()
	defer mq.mu.RUnlock()

	total := 0
	for _, queue := range mq.queues {
		total += len(queue)
	}
	return total
}

// Clear clears the queue for a service
func (mq *MessageQueue) Clear(service string) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	delete(mq.queues, service)
}

// ClearAll clears all queues
func (mq *MessageQueue) ClearAll() {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	mq.queues = make(map[string][]*pb.IPCMessage)
}
