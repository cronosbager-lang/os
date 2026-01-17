package service

import (
	"encoding/json"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics collects and reports service metrics
type Metrics struct {
	serviceName string
	startTime   time.Time
	
	// Counters
	requestCount    int64
	errorCount      int64
	successCount    int64
	
	// Gauges
	activeRequests  int64
	
	// Histograms (simplified)
	requestDurations []time.Duration
	durationMu       sync.Mutex
	maxDurations     int
	
	// Custom metrics
	custom   map[string]interface{}
	customMu sync.RWMutex
}

// NewMetrics creates a new metrics collector
func NewMetrics(serviceName string) *Metrics {
	return &Metrics{
		serviceName:      serviceName,
		startTime:        time.Now(),
		requestDurations: make([]time.Duration, 0, 1000),
		maxDurations:     1000,
		custom:           make(map[string]interface{}),
	}
}

// IncrementRequests increments the request counter
func (m *Metrics) IncrementRequests() {
	atomic.AddInt64(&m.requestCount, 1)
}

// IncrementErrors increments the error counter
func (m *Metrics) IncrementErrors() {
	atomic.AddInt64(&m.errorCount, 1)
}

// IncrementSuccess increments the success counter
func (m *Metrics) IncrementSuccess() {
	atomic.AddInt64(&m.successCount, 1)
}

// IncrementActive increments the active request gauge
func (m *Metrics) IncrementActive() {
	atomic.AddInt64(&m.activeRequests, 1)
}

// DecrementActive decrements the active request gauge
func (m *Metrics) DecrementActive() {
	atomic.AddInt64(&m.activeRequests, -1)
}

// RecordDuration records a request duration
func (m *Metrics) RecordDuration(d time.Duration) {
	m.durationMu.Lock()
	defer m.durationMu.Unlock()
	
	if len(m.requestDurations) >= m.maxDurations {
		// Remove oldest
		m.requestDurations = m.requestDurations[1:]
	}
	m.requestDurations = append(m.requestDurations, d)
}

// SetCustom sets a custom metric
func (m *Metrics) SetCustom(name string, value interface{}) {
	m.customMu.Lock()
	defer m.customMu.Unlock()
	m.custom[name] = value
}

// GetCustom gets a custom metric
func (m *Metrics) GetCustom(name string) (interface{}, bool) {
	m.customMu.RLock()
	defer m.customMu.RUnlock()
	v, ok := m.custom[name]
	return v, ok
}

// IncrementCustom increments a custom counter
func (m *Metrics) IncrementCustom(name string) {
	m.customMu.Lock()
	defer m.customMu.Unlock()
	
	if v, ok := m.custom[name]; ok {
		if count, ok := v.(int64); ok {
			m.custom[name] = count + 1
			return
		}
	}
	m.custom[name] = int64(1)
}

// GetStats returns all metrics as a map
func (m *Metrics) GetStats() map[string]interface{} {
	// Calculate duration stats
	m.durationMu.Lock()
	durations := make([]time.Duration, len(m.requestDurations))
	copy(durations, m.requestDurations)
	m.durationMu.Unlock()
	
	durationStats := m.calculateDurationStats(durations)
	
	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	// Get custom metrics
	m.customMu.RLock()
	custom := make(map[string]interface{})
	for k, v := range m.custom {
		custom[k] = v
	}
	m.customMu.RUnlock()
	
	return map[string]interface{}{
		"service_name":    m.serviceName,
		"uptime_seconds":  time.Since(m.startTime).Seconds(),
		"start_time":      m.startTime,
		
		// Counters
		"request_count":   atomic.LoadInt64(&m.requestCount),
		"error_count":     atomic.LoadInt64(&m.errorCount),
		"success_count":   atomic.LoadInt64(&m.successCount),
		
		// Gauges
		"active_requests": atomic.LoadInt64(&m.activeRequests),
		
		// Duration stats
		"duration_stats":  durationStats,
		
		// Runtime stats
		"runtime": map[string]interface{}{
			"goroutines":   runtime.NumGoroutine(),
			"heap_alloc":   memStats.HeapAlloc,
			"heap_sys":     memStats.HeapSys,
			"heap_objects": memStats.HeapObjects,
			"gc_runs":      memStats.NumGC,
			"gc_pause_ns":  memStats.PauseTotalNs,
		},
		
		// Custom metrics
		"custom": custom,
	}
}

// calculateDurationStats calculates statistics for durations
func (m *Metrics) calculateDurationStats(durations []time.Duration) map[string]interface{} {
	if len(durations) == 0 {
		return map[string]interface{}{
			"count": 0,
		}
	}
	
	// Calculate min, max, avg
	var total time.Duration
	min := durations[0]
	max := durations[0]
	
	for _, d := range durations {
		total += d
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}
	
	avg := total / time.Duration(len(durations))
	
	// Calculate percentiles (simplified)
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	// Simple bubble sort for small datasets
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	p50 := sorted[len(sorted)/2]
	p90 := sorted[int(float64(len(sorted))*0.9)]
	p99 := sorted[int(float64(len(sorted))*0.99)]
	
	return map[string]interface{}{
		"count":      len(durations),
		"min_ms":     float64(min) / float64(time.Millisecond),
		"max_ms":     float64(max) / float64(time.Millisecond),
		"avg_ms":     float64(avg) / float64(time.Millisecond),
		"p50_ms":     float64(p50) / float64(time.Millisecond),
		"p90_ms":     float64(p90) / float64(time.Millisecond),
		"p99_ms":     float64(p99) / float64(time.Millisecond),
		"total_ms":   float64(total) / float64(time.Millisecond),
	}
}

// ToJSON returns metrics as JSON
func (m *Metrics) ToJSON() ([]byte, error) {
	return json.Marshal(m.GetStats())
}

// Reset resets all metrics
func (m *Metrics) Reset() {
	atomic.StoreInt64(&m.requestCount, 0)
	atomic.StoreInt64(&m.errorCount, 0)
	atomic.StoreInt64(&m.successCount, 0)
	atomic.StoreInt64(&m.activeRequests, 0)
	
	m.durationMu.Lock()
	m.requestDurations = make([]time.Duration, 0, m.maxDurations)
	m.durationMu.Unlock()
	
	m.customMu.Lock()
	m.custom = make(map[string]interface{})
	m.customMu.Unlock()
}

// RequestTracker tracks individual requests
type RequestTracker struct {
	metrics   *Metrics
	startTime time.Time
}

// StartRequest starts tracking a request
func (m *Metrics) StartRequest() *RequestTracker {
	m.IncrementRequests()
	m.IncrementActive()
	return &RequestTracker{
		metrics:   m,
		startTime: time.Now(),
	}
}

// End ends tracking and records the duration
func (rt *RequestTracker) End(success bool) {
	duration := time.Since(rt.startTime)
	rt.metrics.RecordDuration(duration)
	rt.metrics.DecrementActive()
	
	if success {
		rt.metrics.IncrementSuccess()
	} else {
		rt.metrics.IncrementErrors()
	}
}

// EndWithError ends tracking with an error
func (rt *RequestTracker) EndWithError() {
	rt.End(false)
}

// EndWithSuccess ends tracking with success
func (rt *RequestTracker) EndWithSuccess() {
	rt.End(true)
}

// ResourceMonitor monitors system resources
type ResourceMonitor struct {
	interval time.Duration
	done     chan struct{}
	
	// Callbacks
	onHighMemory    func(float64)
	onHighCPU       func(float64)
	onHighGoroutines func(int)
	
	// Thresholds
	memoryThreshold    float64
	goroutineThreshold int
	
	mu sync.RWMutex
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(interval time.Duration) *ResourceMonitor {
	return &ResourceMonitor{
		interval:           interval,
		done:               make(chan struct{}),
		memoryThreshold:    80.0, // 80%
		goroutineThreshold: 10000,
	}
}

// SetMemoryThreshold sets the memory usage threshold
func (rm *ResourceMonitor) SetMemoryThreshold(percent float64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.memoryThreshold = percent
}

// SetGoroutineThreshold sets the goroutine count threshold
func (rm *ResourceMonitor) SetGoroutineThreshold(count int) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.goroutineThreshold = count
}

// OnHighMemory sets the callback for high memory usage
func (rm *ResourceMonitor) OnHighMemory(callback func(float64)) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.onHighMemory = callback
}

// OnHighGoroutines sets the callback for high goroutine count
func (rm *ResourceMonitor) OnHighGoroutines(callback func(int)) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.onHighGoroutines = callback
}

// Start starts the resource monitor
func (rm *ResourceMonitor) Start() {
	go rm.monitorLoop()
}

// Stop stops the resource monitor
func (rm *ResourceMonitor) Stop() {
	close(rm.done)
}

// monitorLoop runs the monitoring loop
func (rm *ResourceMonitor) monitorLoop() {
	ticker := time.NewTicker(rm.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-rm.done:
			return
		case <-ticker.C:
			rm.checkResources()
		}
	}
}

// checkResources checks resource usage
func (rm *ResourceMonitor) checkResources() {
	rm.mu.RLock()
	memThreshold := rm.memoryThreshold
	goroutineThreshold := rm.goroutineThreshold
	onHighMemory := rm.onHighMemory
	onHighGoroutines := rm.onHighGoroutines
	rm.mu.RUnlock()
	
	// Check memory
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	memUsage := float64(memStats.Alloc) / float64(memStats.Sys) * 100
	
	if memUsage > memThreshold && onHighMemory != nil {
		onHighMemory(memUsage)
	}
	
	// Check goroutines
	goroutineCount := runtime.NumGoroutine()
	if goroutineCount > goroutineThreshold && onHighGoroutines != nil {
		onHighGoroutines(goroutineCount)
	}
}

// GetResourceStats returns current resource statistics
func (rm *ResourceMonitor) GetResourceStats() map[string]interface{} {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	return map[string]interface{}{
		"memory": map[string]interface{}{
			"alloc_bytes":      memStats.Alloc,
			"total_alloc":      memStats.TotalAlloc,
			"sys_bytes":        memStats.Sys,
			"heap_alloc":       memStats.HeapAlloc,
			"heap_sys":         memStats.HeapSys,
			"heap_idle":        memStats.HeapIdle,
			"heap_inuse":       memStats.HeapInuse,
			"heap_released":    memStats.HeapReleased,
			"heap_objects":     memStats.HeapObjects,
			"stack_inuse":      memStats.StackInuse,
			"stack_sys":        memStats.StackSys,
			"usage_percent":    float64(memStats.Alloc) / float64(memStats.Sys) * 100,
		},
		"gc": map[string]interface{}{
			"num_gc":           memStats.NumGC,
			"pause_total_ns":   memStats.PauseTotalNs,
			"last_gc":          time.Unix(0, int64(memStats.LastGC)),
			"gc_cpu_fraction":  memStats.GCCPUFraction,
		},
		"goroutines": runtime.NumGoroutine(),
		"num_cpu":    runtime.NumCPU(),
		"gomaxprocs": runtime.GOMAXPROCS(0),
	}
}
