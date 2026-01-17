package registry

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// LoadBalancer provides load balancing across service instances
type LoadBalancer interface {
	// Select selects a service instance from the available instances
	Select(services []*ServiceInfo) *ServiceInfo
}

// RoundRobinBalancer implements round-robin load balancing
type RoundRobinBalancer struct {
	counter uint64
}

// NewRoundRobinBalancer creates a new round-robin load balancer
func NewRoundRobinBalancer() *RoundRobinBalancer {
	return &RoundRobinBalancer{}
}

// Select selects the next service in round-robin order
func (b *RoundRobinBalancer) Select(services []*ServiceInfo) *ServiceInfo {
	if len(services) == 0 {
		return nil
	}
	
	idx := atomic.AddUint64(&b.counter, 1) % uint64(len(services))
	return services[idx]
}

// RandomBalancer implements random load balancing
type RandomBalancer struct {
	rng *rand.Rand
	mu  sync.Mutex
}

// NewRandomBalancer creates a new random load balancer
func NewRandomBalancer() *RandomBalancer {
	return &RandomBalancer{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Select selects a random service
func (b *RandomBalancer) Select(services []*ServiceInfo) *ServiceInfo {
	if len(services) == 0 {
		return nil
	}
	
	b.mu.Lock()
	idx := b.rng.Intn(len(services))
	b.mu.Unlock()
	
	return services[idx]
}

// WeightedBalancer implements weighted load balancing
type WeightedBalancer struct {
	weights map[string]int // service key -> weight
	mu      sync.RWMutex
	rng     *rand.Rand
}

// NewWeightedBalancer creates a new weighted load balancer
func NewWeightedBalancer() *WeightedBalancer {
	return &WeightedBalancer{
		weights: make(map[string]int),
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SetWeight sets the weight for a service
func (b *WeightedBalancer) SetWeight(namespace, name string, weight int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.weights[namespace+"/"+name] = weight
}

// Select selects a service based on weights
func (b *WeightedBalancer) Select(services []*ServiceInfo) *ServiceInfo {
	if len(services) == 0 {
		return nil
	}
	
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	// Calculate total weight
	totalWeight := 0
	for _, svc := range services {
		key := svc.Namespace + "/" + svc.Name
		weight := b.weights[key]
		if weight <= 0 {
			weight = 1 // Default weight
		}
		totalWeight += weight
	}
	
	// Select based on weight
	target := b.rng.Intn(totalWeight)
	current := 0
	
	for _, svc := range services {
		key := svc.Namespace + "/" + svc.Name
		weight := b.weights[key]
		if weight <= 0 {
			weight = 1
		}
		current += weight
		if current > target {
			return svc
		}
	}
	
	return services[0]
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	state       CircuitState
	failures    int
	successes   int
	lastFailure time.Time
	
	// Configuration
	failureThreshold int
	successThreshold int
	timeout          time.Duration
	
	mu sync.RWMutex
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            CircuitClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
	}
}

// Allow checks if a request should be allowed
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = CircuitHalfOpen
			cb.successes = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	}
	
	return false
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.failures = 0
	
	if cb.state == CircuitHalfOpen {
		cb.successes++
		if cb.successes >= cb.successThreshold {
			cb.state = CircuitClosed
		}
	}
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.failures++
	cb.lastFailure = time.Now()
	
	if cb.failures >= cb.failureThreshold {
		cb.state = CircuitOpen
	}
}

// State returns the current circuit state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// ServiceDiscovery provides service discovery with load balancing
type ServiceDiscovery struct {
	registry *Registry
	balancer LoadBalancer
	breakers map[string]*CircuitBreaker
	mu       sync.RWMutex
	
	// Circuit breaker config
	failureThreshold int
	successThreshold int
	breakerTimeout   time.Duration
}

// NewServiceDiscovery creates a new service discovery instance
func NewServiceDiscovery(registry *Registry, balancer LoadBalancer) *ServiceDiscovery {
	if balancer == nil {
		balancer = NewRoundRobinBalancer()
	}
	
	return &ServiceDiscovery{
		registry:         registry,
		balancer:         balancer,
		breakers:         make(map[string]*CircuitBreaker),
		failureThreshold: 5,
		successThreshold: 2,
		breakerTimeout:   30 * time.Second,
	}
}

// Discover finds a healthy service instance
func (sd *ServiceDiscovery) Discover(name string) (*ServiceInfo, error) {
	return sd.DiscoverInNamespace("default", name)
}

// DiscoverInNamespace finds a healthy service instance in a specific namespace
func (sd *ServiceDiscovery) DiscoverInNamespace(namespace, name string) (*ServiceInfo, error) {
	// Get all instances
	services := sd.registry.Query().
		InNamespace(namespace).
		OnlyHealthy().
		Execute()
	
	// Filter by name
	var matching []*ServiceInfo
	for _, svc := range services {
		if svc.Name == name {
			// Check circuit breaker
			if sd.isAllowed(svc) {
				matching = append(matching, svc)
			}
		}
	}
	
	if len(matching) == 0 {
		return nil, ErrNoHealthyInstances
	}
	
	return sd.balancer.Select(matching), nil
}

// DiscoverByCapability finds a service with a specific capability
func (sd *ServiceDiscovery) DiscoverByCapability(capability string) (*ServiceInfo, error) {
	services := sd.registry.Query().
		WithCapability(capability).
		OnlyHealthy().
		Execute()
	
	// Filter by circuit breaker
	var allowed []*ServiceInfo
	for _, svc := range services {
		if sd.isAllowed(svc) {
			allowed = append(allowed, svc)
		}
	}
	
	if len(allowed) == 0 {
		return nil, ErrNoHealthyInstances
	}
	
	return sd.balancer.Select(allowed), nil
}

// DiscoverAll finds all healthy instances of a service
func (sd *ServiceDiscovery) DiscoverAll(name string) []*ServiceInfo {
	services := sd.registry.Query().
		OnlyHealthy().
		Execute()
	
	var matching []*ServiceInfo
	for _, svc := range services {
		if svc.Name == name && sd.isAllowed(svc) {
			matching = append(matching, svc)
		}
	}
	
	return matching
}

// RecordSuccess records a successful call to a service
func (sd *ServiceDiscovery) RecordSuccess(svc *ServiceInfo) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	
	key := svc.Namespace + "/" + svc.Name
	if breaker, ok := sd.breakers[key]; ok {
		breaker.RecordSuccess()
	}
}

// RecordFailure records a failed call to a service
func (sd *ServiceDiscovery) RecordFailure(svc *ServiceInfo) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	
	key := svc.Namespace + "/" + svc.Name
	breaker, ok := sd.breakers[key]
	if !ok {
		breaker = NewCircuitBreaker(sd.failureThreshold, sd.successThreshold, sd.breakerTimeout)
		sd.breakers[key] = breaker
	}
	breaker.RecordFailure()
}

// isAllowed checks if a service is allowed by its circuit breaker
func (sd *ServiceDiscovery) isAllowed(svc *ServiceInfo) bool {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	
	key := svc.Namespace + "/" + svc.Name
	breaker, ok := sd.breakers[key]
	if !ok {
		return true // No breaker = allowed
	}
	return breaker.Allow()
}

// GetCircuitState returns the circuit breaker state for a service
func (sd *ServiceDiscovery) GetCircuitState(namespace, name string) CircuitState {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	
	key := namespace + "/" + name
	breaker, ok := sd.breakers[key]
	if !ok {
		return CircuitClosed
	}
	return breaker.State()
}

// Errors
var (
	ErrNoHealthyInstances = &DiscoveryError{Message: "no healthy service instances available"}
)

// DiscoveryError represents a service discovery error
type DiscoveryError struct {
	Message string
}

func (e *DiscoveryError) Error() string {
	return e.Message
}

// Watch provides a way to watch for service changes
type Watch struct {
	registry *Registry
	filter   func(*ServiceInfo) bool
	events   chan WatchEvent
	done     chan struct{}
}

// WatchEvent represents a service change event
type WatchEvent struct {
	Type    WatchEventType
	Service *ServiceInfo
}

// WatchEventType represents the type of watch event
type WatchEventType int

const (
	WatchEventAdded WatchEventType = iota
	WatchEventRemoved
	WatchEventModified
)

// NewWatch creates a new watch
func (r *Registry) Watch(filter func(*ServiceInfo) bool) *Watch {
	w := &Watch{
		registry: r,
		filter:   filter,
		events:   make(chan WatchEvent, 100),
		done:     make(chan struct{}),
	}
	
	// Register callbacks
	r.OnRegister(func(info *ServiceInfo) {
		if filter == nil || filter(info) {
			select {
			case w.events <- WatchEvent{Type: WatchEventAdded, Service: info}:
			default:
			}
		}
	})
	
	r.OnDeregister(func(info *ServiceInfo) {
		if filter == nil || filter(info) {
			select {
			case w.events <- WatchEvent{Type: WatchEventRemoved, Service: info}:
			default:
			}
		}
	})
	
	r.OnStatusChange(func(info *ServiceInfo, old, new ServiceStatus) {
		if filter == nil || filter(info) {
			select {
			case w.events <- WatchEvent{Type: WatchEventModified, Service: info}:
			default:
			}
		}
	})
	
	return w
}

// Events returns the event channel
func (w *Watch) Events() <-chan WatchEvent {
	return w.events
}

// Stop stops the watch
func (w *Watch) Stop() {
	close(w.done)
	close(w.events)
}
