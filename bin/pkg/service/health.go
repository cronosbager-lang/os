package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

// HealthStatus represents the health status of a service
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthCheckResult contains the result of a health check
type HealthCheckResult struct {
	Status    HealthStatus          `json:"status"`
	Message   string                `json:"message,omitempty"`
	Timestamp time.Time             `json:"timestamp"`
	Duration  time.Duration         `json:"duration"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// HealthCheck defines a health check function
type HealthCheck func(ctx context.Context) HealthCheckResult

// LivenessCheck checks if the service is alive
type LivenessCheck func(ctx context.Context) bool

// ReadinessCheck checks if the service is ready to accept traffic
type ReadinessCheck func(ctx context.Context) bool

// HealthChecker manages health checks for a service
type HealthChecker struct {
	serviceName string
	checks      map[string]HealthCheck
	liveness    LivenessCheck
	readiness   ReadinessCheck
	
	// Results
	lastResults map[string]HealthCheckResult
	lastLiveness bool
	lastReadiness bool
	
	// Configuration
	checkInterval time.Duration
	checkTimeout  time.Duration
	
	// State
	mu      sync.RWMutex
	running bool
	done    chan struct{}
	
	// Callbacks
	onHealthChange []func(string, HealthCheckResult)
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(serviceName string) *HealthChecker {
	return &HealthChecker{
		serviceName:   serviceName,
		checks:        make(map[string]HealthCheck),
		lastResults:   make(map[string]HealthCheckResult),
		checkInterval: 30 * time.Second,
		checkTimeout:  10 * time.Second,
		done:          make(chan struct{}),
		lastLiveness:  true,
		lastReadiness: true,
	}
}

// RegisterCheck registers a named health check
func (hc *HealthChecker) RegisterCheck(name string, check HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checks[name] = check
}

// SetLivenessCheck sets the liveness check
func (hc *HealthChecker) SetLivenessCheck(check LivenessCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.liveness = check
}

// SetReadinessCheck sets the readiness check
func (hc *HealthChecker) SetReadinessCheck(check ReadinessCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.readiness = check
}

// SetCheckInterval sets the interval between health checks
func (hc *HealthChecker) SetCheckInterval(interval time.Duration) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checkInterval = interval
}

// SetCheckTimeout sets the timeout for health checks
func (hc *HealthChecker) SetCheckTimeout(timeout time.Duration) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checkTimeout = timeout
}

// OnHealthChange registers a callback for health changes
func (hc *HealthChecker) OnHealthChange(callback func(string, HealthCheckResult)) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.onHealthChange = append(hc.onHealthChange, callback)
}

// Start starts the health check loop
func (hc *HealthChecker) Start() {
	hc.mu.Lock()
	if hc.running {
		hc.mu.Unlock()
		return
	}
	hc.running = true
	hc.mu.Unlock()

	go hc.checkLoop()
}

// Stop stops the health check loop
func (hc *HealthChecker) Stop() {
	hc.mu.Lock()
	if !hc.running {
		hc.mu.Unlock()
		return
	}
	hc.running = false
	hc.mu.Unlock()

	close(hc.done)
}

// checkLoop runs health checks periodically
func (hc *HealthChecker) checkLoop() {
	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	// Run initial check
	hc.runAllChecks()

	for {
		select {
		case <-hc.done:
			return
		case <-ticker.C:
			hc.runAllChecks()
		}
	}
}

// runAllChecks runs all registered health checks
func (hc *HealthChecker) runAllChecks() {
	hc.mu.RLock()
	checks := make(map[string]HealthCheck)
	for name, check := range hc.checks {
		checks[name] = check
	}
	timeout := hc.checkTimeout
	hc.mu.RUnlock()

	for name, check := range checks {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		result := hc.runCheck(ctx, name, check)
		cancel()

		hc.mu.Lock()
		oldResult, exists := hc.lastResults[name]
		hc.lastResults[name] = result
		
		// Notify if status changed
		if !exists || oldResult.Status != result.Status {
			for _, callback := range hc.onHealthChange {
				go callback(name, result)
			}
		}
		hc.mu.Unlock()
	}
}

// runCheck runs a single health check
func (hc *HealthChecker) runCheck(ctx context.Context, name string, check HealthCheck) HealthCheckResult {
	start := time.Now()
	
	// Run check with panic recovery
	resultCh := make(chan HealthCheckResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultCh <- HealthCheckResult{
					Status:    HealthStatusUnhealthy,
					Message:   fmt.Sprintf("panic: %v", r),
					Timestamp: time.Now(),
					Duration:  time.Since(start),
				}
			}
		}()
		resultCh <- check(ctx)
	}()

	select {
	case result := <-resultCh:
		result.Duration = time.Since(start)
		result.Timestamp = time.Now()
		return result
	case <-ctx.Done():
		return HealthCheckResult{
			Status:    HealthStatusUnhealthy,
			Message:   "health check timeout",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}
	}
}

// CheckHealth runs all health checks and returns the overall status
func (hc *HealthChecker) CheckHealth(ctx context.Context) HealthCheckResult {
	hc.mu.RLock()
	checks := make(map[string]HealthCheck)
	for name, check := range hc.checks {
		checks[name] = check
	}
	timeout := hc.checkTimeout
	hc.mu.RUnlock()

	results := make(map[string]HealthCheckResult)
	overallStatus := HealthStatusHealthy
	var messages []string

	for name, check := range checks {
		checkCtx, cancel := context.WithTimeout(ctx, timeout)
		result := hc.runCheck(checkCtx, name, check)
		cancel()

		results[name] = result

		switch result.Status {
		case HealthStatusUnhealthy:
			overallStatus = HealthStatusUnhealthy
			messages = append(messages, fmt.Sprintf("%s: %s", name, result.Message))
		case HealthStatusDegraded:
			if overallStatus == HealthStatusHealthy {
				overallStatus = HealthStatusDegraded
			}
			messages = append(messages, fmt.Sprintf("%s: %s", name, result.Message))
		}
	}

	message := ""
	if len(messages) > 0 {
		message = fmt.Sprintf("%d issues: %v", len(messages), messages)
	}

	details := make(map[string]interface{})
	for name, result := range results {
		details[name] = result
	}

	return HealthCheckResult{
		Status:    overallStatus,
		Message:   message,
		Timestamp: time.Now(),
		Details:   details,
	}
}

// IsLive checks if the service is alive
func (hc *HealthChecker) IsLive(ctx context.Context) bool {
	hc.mu.RLock()
	check := hc.liveness
	hc.mu.RUnlock()

	if check == nil {
		return true // Default to live if no check defined
	}

	result := check(ctx)
	
	hc.mu.Lock()
	hc.lastLiveness = result
	hc.mu.Unlock()

	return result
}

// IsReady checks if the service is ready
func (hc *HealthChecker) IsReady(ctx context.Context) bool {
	hc.mu.RLock()
	check := hc.readiness
	hc.mu.RUnlock()

	if check == nil {
		return true // Default to ready if no check defined
	}

	result := check(ctx)
	
	hc.mu.Lock()
	hc.lastReadiness = result
	hc.mu.Unlock()

	return result
}

// GetLastResults returns the last health check results
func (hc *HealthChecker) GetLastResults() map[string]HealthCheckResult {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	results := make(map[string]HealthCheckResult)
	for name, result := range hc.lastResults {
		results[name] = result
	}
	return results
}

// GetLastResult returns the last result for a specific check
func (hc *HealthChecker) GetLastResult(name string) (HealthCheckResult, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result, ok := hc.lastResults[name]
	return result, ok
}

// Common health checks

// MemoryCheck creates a health check for memory usage
func MemoryCheck(maxUsagePercent float64) HealthCheck {
	return func(ctx context.Context) HealthCheckResult {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		// Get system memory info (simplified)
		totalMemory := m.Sys
		usedMemory := m.Alloc
		usagePercent := float64(usedMemory) / float64(totalMemory) * 100

		status := HealthStatusHealthy
		message := ""

		if usagePercent > maxUsagePercent {
			status = HealthStatusDegraded
			message = fmt.Sprintf("memory usage %.1f%% exceeds threshold %.1f%%", usagePercent, maxUsagePercent)
		}

		return HealthCheckResult{
			Status:  status,
			Message: message,
			Details: map[string]interface{}{
				"total_bytes":   totalMemory,
				"used_bytes":    usedMemory,
				"usage_percent": usagePercent,
				"heap_alloc":    m.HeapAlloc,
				"heap_sys":      m.HeapSys,
				"num_gc":        m.NumGC,
			},
		}
	}
}

// GoroutineCheck creates a health check for goroutine count
func GoroutineCheck(maxGoroutines int) HealthCheck {
	return func(ctx context.Context) HealthCheckResult {
		count := runtime.NumGoroutine()

		status := HealthStatusHealthy
		message := ""

		if count > maxGoroutines {
			status = HealthStatusDegraded
			message = fmt.Sprintf("goroutine count %d exceeds threshold %d", count, maxGoroutines)
		}

		return HealthCheckResult{
			Status:  status,
			Message: message,
			Details: map[string]interface{}{
				"goroutine_count": count,
				"max_goroutines":  maxGoroutines,
			},
		}
	}
}

// DiskCheck creates a health check for disk usage
func DiskCheck(path string, maxUsagePercent float64) HealthCheck {
	return func(ctx context.Context) HealthCheckResult {
		// Get disk usage (simplified - would need syscall for real implementation)
		var stat struct {
			Total uint64
			Free  uint64
		}
		
		// Placeholder - in real implementation, use syscall.Statfs
		stat.Total = 100 * 1024 * 1024 * 1024 // 100GB
		stat.Free = 50 * 1024 * 1024 * 1024   // 50GB

		usedPercent := float64(stat.Total-stat.Free) / float64(stat.Total) * 100

		status := HealthStatusHealthy
		message := ""

		if usedPercent > maxUsagePercent {
			status = HealthStatusDegraded
			message = fmt.Sprintf("disk usage %.1f%% exceeds threshold %.1f%%", usedPercent, maxUsagePercent)
		}

		return HealthCheckResult{
			Status:  status,
			Message: message,
			Details: map[string]interface{}{
				"path":          path,
				"total_bytes":   stat.Total,
				"free_bytes":    stat.Free,
				"usage_percent": usedPercent,
			},
		}
	}
}

// FileExistsCheck creates a health check for file existence
func FileExistsCheck(path string) HealthCheck {
	return func(ctx context.Context) HealthCheckResult {
		_, err := os.Stat(path)
		
		if err != nil {
			return HealthCheckResult{
				Status:  HealthStatusUnhealthy,
				Message: fmt.Sprintf("file not found: %s", path),
				Details: map[string]interface{}{
					"path":  path,
					"error": err.Error(),
				},
			}
		}

		return HealthCheckResult{
			Status: HealthStatusHealthy,
			Details: map[string]interface{}{
				"path": path,
			},
		}
	}
}

// CustomCheck creates a custom health check from a function
func CustomCheck(name string, check func() error) HealthCheck {
	return func(ctx context.Context) HealthCheckResult {
		err := check()
		
		if err != nil {
			return HealthCheckResult{
				Status:  HealthStatusUnhealthy,
				Message: err.Error(),
			}
		}

		return HealthCheckResult{
			Status: HealthStatusHealthy,
		}
	}
}

// HealthEndpoint provides HTTP-style health endpoint responses
type HealthEndpoint struct {
	checker *HealthChecker
}

// NewHealthEndpoint creates a new health endpoint
func NewHealthEndpoint(checker *HealthChecker) *HealthEndpoint {
	return &HealthEndpoint{checker: checker}
}

// Health returns the overall health status as JSON
func (he *HealthEndpoint) Health(ctx context.Context) ([]byte, error) {
	result := he.checker.CheckHealth(ctx)
	return json.Marshal(result)
}

// Live returns the liveness status
func (he *HealthEndpoint) Live(ctx context.Context) ([]byte, error) {
	live := he.checker.IsLive(ctx)
	return json.Marshal(map[string]interface{}{
		"live":      live,
		"timestamp": time.Now(),
	})
}

// Ready returns the readiness status
func (he *HealthEndpoint) Ready(ctx context.Context) ([]byte, error) {
	ready := he.checker.IsReady(ctx)
	return json.Marshal(map[string]interface{}{
		"ready":     ready,
		"timestamp": time.Now(),
	})
}
