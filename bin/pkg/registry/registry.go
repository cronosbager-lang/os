package registry

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ServiceType represents the type of service
type ServiceType string

const (
	ServiceTypeCore   ServiceType = "core"
	ServiceTypeAgent  ServiceType = "agent"
	ServiceTypeUser   ServiceType = "user"
	ServiceTypeSystem ServiceType = "system"
)

// ServiceStatus represents the current status of a service
type ServiceStatus string

const (
	StatusUnknown     ServiceStatus = "unknown"
	StatusStarting    ServiceStatus = "starting"
	StatusRunning     ServiceStatus = "running"
	StatusStopping    ServiceStatus = "stopping"
	StatusStopped     ServiceStatus = "stopped"
	StatusFailed      ServiceStatus = "failed"
	StatusDegraded    ServiceStatus = "degraded"
)

// Capability represents a service capability
type Capability struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description,omitempty"`
	Methods     []string          `json:"methods,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ServiceInfo contains information about a registered service
type ServiceInfo struct {
	Name         string            `json:"name"`
	Type         ServiceType       `json:"type"`
	Namespace    string            `json:"namespace"`
	Version      string            `json:"version"`
	Description  string            `json:"description,omitempty"`
	Status       ServiceStatus     `json:"status"`
	Capabilities []Capability      `json:"capabilities,omitempty"`
	Endpoints    []Endpoint        `json:"endpoints,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	
	// Runtime info
	PID          int               `json:"pid,omitempty"`
	RegisteredAt time.Time         `json:"registered_at"`
	LastSeen     time.Time         `json:"last_seen"`
	Healthy      bool              `json:"healthy"`
	
	// Health check info
	HealthCheck  *HealthCheckConfig `json:"health_check,omitempty"`
}

// Endpoint represents a service endpoint
type Endpoint struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"` // "ipc", "http", "grpc"
	Address  string `json:"address"`
}

// HealthCheckConfig contains health check configuration
type HealthCheckConfig struct {
	Enabled  bool          `json:"enabled"`
	Interval time.Duration `json:"interval"`
	Timeout  time.Duration `json:"timeout"`
	Method   string        `json:"method"` // IPC method to call
}

// Registry manages service registration and discovery
type Registry struct {
	services   map[string]*ServiceInfo
	namespaces map[string]map[string]*ServiceInfo // namespace -> name -> service
	byCapability map[string][]*ServiceInfo        // capability -> services
	mu         sync.RWMutex
	
	// Event callbacks
	onRegister   []func(*ServiceInfo)
	onDeregister []func(*ServiceInfo)
	onStatusChange []func(*ServiceInfo, ServiceStatus, ServiceStatus)
}

// NewRegistry creates a new service registry
func NewRegistry() *Registry {
	return &Registry{
		services:     make(map[string]*ServiceInfo),
		namespaces:   make(map[string]map[string]*ServiceInfo),
		byCapability: make(map[string][]*ServiceInfo),
	}
}

// Register registers a service with the registry
func (r *Registry) Register(info *ServiceInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate
	if info.Name == "" {
		return fmt.Errorf("service name is required")
	}

	// Set defaults
	if info.Namespace == "" {
		info.Namespace = "default"
	}
	if info.Type == "" {
		info.Type = ServiceTypeUser
	}
	if info.Status == "" {
		info.Status = StatusRunning
	}
	info.RegisteredAt = time.Now()
	info.LastSeen = time.Now()
	info.Healthy = true

	// Create namespace if needed
	if _, ok := r.namespaces[info.Namespace]; !ok {
		r.namespaces[info.Namespace] = make(map[string]*ServiceInfo)
	}

	// Check for existing service
	key := r.serviceKey(info.Namespace, info.Name)
	if existing, ok := r.services[key]; ok {
		// Update existing service
		existing.Status = info.Status
		existing.LastSeen = time.Now()
		existing.Healthy = true
		existing.PID = info.PID
		existing.Capabilities = info.Capabilities
		existing.Endpoints = info.Endpoints
		existing.Metadata = info.Metadata
		return nil
	}

	// Register new service
	r.services[key] = info
	r.namespaces[info.Namespace][info.Name] = info

	// Index by capability
	for _, cap := range info.Capabilities {
		r.byCapability[cap.Name] = append(r.byCapability[cap.Name], info)
	}

	// Notify listeners
	for _, callback := range r.onRegister {
		go callback(info)
	}

	return nil
}

// Deregister removes a service from the registry
func (r *Registry) Deregister(namespace, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if namespace == "" {
		namespace = "default"
	}

	key := r.serviceKey(namespace, name)
	info, ok := r.services[key]
	if !ok {
		return fmt.Errorf("service not found: %s/%s", namespace, name)
	}

	// Remove from main map
	delete(r.services, key)

	// Remove from namespace
	if ns, ok := r.namespaces[namespace]; ok {
		delete(ns, name)
	}

	// Remove from capability index
	for _, cap := range info.Capabilities {
		services := r.byCapability[cap.Name]
		for i, svc := range services {
			if svc.Name == name && svc.Namespace == namespace {
				r.byCapability[cap.Name] = append(services[:i], services[i+1:]...)
				break
			}
		}
	}

	// Notify listeners
	for _, callback := range r.onDeregister {
		go callback(info)
	}

	return nil
}

// Get retrieves a service by namespace and name
func (r *Registry) Get(namespace, name string) (*ServiceInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if namespace == "" {
		namespace = "default"
	}

	key := r.serviceKey(namespace, name)
	info, ok := r.services[key]
	return info, ok
}

// GetByName retrieves a service by name (searches all namespaces)
func (r *Registry) GetByName(name string) (*ServiceInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// First check default namespace
	if info, ok := r.namespaces["default"][name]; ok {
		return info, true
	}

	// Search all namespaces
	for _, ns := range r.namespaces {
		if info, ok := ns[name]; ok {
			return info, true
		}
	}

	return nil, false
}

// List returns all registered services
func (r *Registry) List() []*ServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]*ServiceInfo, 0, len(r.services))
	for _, info := range r.services {
		services = append(services, info)
	}
	return services
}

// ListByNamespace returns services in a specific namespace
func (r *Registry) ListByNamespace(namespace string) []*ServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ns, ok := r.namespaces[namespace]
	if !ok {
		return nil
	}

	services := make([]*ServiceInfo, 0, len(ns))
	for _, info := range ns {
		services = append(services, info)
	}
	return services
}

// ListByType returns services of a specific type
func (r *Registry) ListByType(serviceType ServiceType) []*ServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var services []*ServiceInfo
	for _, info := range r.services {
		if info.Type == serviceType {
			services = append(services, info)
		}
	}
	return services
}

// ListByCapability returns services with a specific capability
func (r *Registry) ListByCapability(capability string) []*ServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := r.byCapability[capability]
	result := make([]*ServiceInfo, len(services))
	copy(result, services)
	return result
}

// ListByTag returns services with a specific tag
func (r *Registry) ListByTag(tag string) []*ServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var services []*ServiceInfo
	for _, info := range r.services {
		for _, t := range info.Tags {
			if t == tag {
				services = append(services, info)
				break
			}
		}
	}
	return services
}

// ListHealthy returns only healthy services
func (r *Registry) ListHealthy() []*ServiceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var services []*ServiceInfo
	for _, info := range r.services {
		if info.Healthy && info.Status == StatusRunning {
			services = append(services, info)
		}
	}
	return services
}

// UpdateStatus updates the status of a service
func (r *Registry) UpdateStatus(namespace, name string, status ServiceStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if namespace == "" {
		namespace = "default"
	}

	key := r.serviceKey(namespace, name)
	info, ok := r.services[key]
	if !ok {
		return fmt.Errorf("service not found: %s/%s", namespace, name)
	}

	oldStatus := info.Status
	info.Status = status
	info.LastSeen = time.Now()

	// Update healthy flag based on status
	info.Healthy = status == StatusRunning

	// Notify listeners
	if oldStatus != status {
		for _, callback := range r.onStatusChange {
			go callback(info, oldStatus, status)
		}
	}

	return nil
}

// UpdateHealth updates the health status of a service
func (r *Registry) UpdateHealth(namespace, name string, healthy bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if namespace == "" {
		namespace = "default"
	}

	key := r.serviceKey(namespace, name)
	info, ok := r.services[key]
	if !ok {
		return fmt.Errorf("service not found: %s/%s", namespace, name)
	}

	info.Healthy = healthy
	info.LastSeen = time.Now()

	if !healthy && info.Status == StatusRunning {
		info.Status = StatusDegraded
	} else if healthy && info.Status == StatusDegraded {
		info.Status = StatusRunning
	}

	return nil
}

// Heartbeat updates the last seen time for a service
func (r *Registry) Heartbeat(namespace, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if namespace == "" {
		namespace = "default"
	}

	key := r.serviceKey(namespace, name)
	info, ok := r.services[key]
	if !ok {
		return fmt.Errorf("service not found: %s/%s", namespace, name)
	}

	info.LastSeen = time.Now()
	return nil
}

// OnRegister registers a callback for service registration
func (r *Registry) OnRegister(callback func(*ServiceInfo)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onRegister = append(r.onRegister, callback)
}

// OnDeregister registers a callback for service deregistration
func (r *Registry) OnDeregister(callback func(*ServiceInfo)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onDeregister = append(r.onDeregister, callback)
}

// OnStatusChange registers a callback for status changes
func (r *Registry) OnStatusChange(callback func(*ServiceInfo, ServiceStatus, ServiceStatus)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onStatusChange = append(r.onStatusChange, callback)
}

// serviceKey generates a unique key for a service
func (r *Registry) serviceKey(namespace, name string) string {
	return namespace + "/" + name
}

// Stats returns registry statistics
func (r *Registry) Stats() RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := RegistryStats{
		TotalServices:   len(r.services),
		HealthyServices: 0,
		ByType:          make(map[ServiceType]int),
		ByNamespace:     make(map[string]int),
		ByStatus:        make(map[ServiceStatus]int),
	}

	for _, info := range r.services {
		if info.Healthy {
			stats.HealthyServices++
		}
		stats.ByType[info.Type]++
		stats.ByNamespace[info.Namespace]++
		stats.ByStatus[info.Status]++
	}

	return stats
}

// RegistryStats contains registry statistics
type RegistryStats struct {
	TotalServices   int                    `json:"total_services"`
	HealthyServices int                    `json:"healthy_services"`
	ByType          map[ServiceType]int    `json:"by_type"`
	ByNamespace     map[string]int         `json:"by_namespace"`
	ByStatus        map[ServiceStatus]int  `json:"by_status"`
}

// ToJSON serializes the registry to JSON
func (r *Registry) ToJSON() ([]byte, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return json.Marshal(r.services)
}

// Query provides a fluent interface for querying services
type Query struct {
	registry  *Registry
	namespace string
	serviceType ServiceType
	capability string
	tag       string
	healthy   *bool
	status    ServiceStatus
}

// NewQuery creates a new query
func (r *Registry) Query() *Query {
	return &Query{registry: r}
}

// InNamespace filters by namespace
func (q *Query) InNamespace(namespace string) *Query {
	q.namespace = namespace
	return q
}

// OfType filters by service type
func (q *Query) OfType(serviceType ServiceType) *Query {
	q.serviceType = serviceType
	return q
}

// WithCapability filters by capability
func (q *Query) WithCapability(capability string) *Query {
	q.capability = capability
	return q
}

// WithTag filters by tag
func (q *Query) WithTag(tag string) *Query {
	q.tag = tag
	return q
}

// OnlyHealthy filters to only healthy services
func (q *Query) OnlyHealthy() *Query {
	healthy := true
	q.healthy = &healthy
	return q
}

// WithStatus filters by status
func (q *Query) WithStatus(status ServiceStatus) *Query {
	q.status = status
	return q
}

// Execute runs the query and returns matching services
func (q *Query) Execute() []*ServiceInfo {
	q.registry.mu.RLock()
	defer q.registry.mu.RUnlock()

	var results []*ServiceInfo

	for _, info := range q.registry.services {
		if q.matches(info) {
			results = append(results, info)
		}
	}

	return results
}

// First returns the first matching service
func (q *Query) First() (*ServiceInfo, bool) {
	results := q.Execute()
	if len(results) > 0 {
		return results[0], true
	}
	return nil, false
}

// matches checks if a service matches the query criteria
func (q *Query) matches(info *ServiceInfo) bool {
	if q.namespace != "" && info.Namespace != q.namespace {
		return false
	}
	if q.serviceType != "" && info.Type != q.serviceType {
		return false
	}
	if q.status != "" && info.Status != q.status {
		return false
	}
	if q.healthy != nil && info.Healthy != *q.healthy {
		return false
	}
	if q.capability != "" {
		found := false
		for _, cap := range info.Capabilities {
			if cap.Name == q.capability {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if q.tag != "" {
		found := false
		for _, t := range info.Tags {
			if t == q.tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
