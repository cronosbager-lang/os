package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// GarbageCollector handles garbage collection for the store
type GarbageCollector struct {
	store *Store
	
	// Configuration
	dryRun       bool
	minAge       time.Duration
	keepRecent   int
	
	// State
	running      bool
	mu           sync.Mutex
	
	// Callbacks
	onProgress   func(GCProgress)
	onComplete   func(GCResult)
}

// GCProgress contains progress information during GC
type GCProgress struct {
	Phase           string `json:"phase"`
	ObjectsScanned  int64  `json:"objects_scanned"`
	ObjectsMarked   int64  `json:"objects_marked"`
	ObjectsDeleted  int64  `json:"objects_deleted"`
	BytesFreed      int64  `json:"bytes_freed"`
}

// GCResult contains the result of garbage collection
type GCResult struct {
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time"`
	Duration       time.Duration `json:"duration"`
	ObjectsScanned int64         `json:"objects_scanned"`
	ObjectsDeleted int64         `json:"objects_deleted"`
	BytesFreed     int64         `json:"bytes_freed"`
	Errors         []string      `json:"errors,omitempty"`
	DryRun         bool          `json:"dry_run"`
}

// NewGarbageCollector creates a new garbage collector
func NewGarbageCollector(store *Store) *GarbageCollector {
	return &GarbageCollector{
		store:      store,
		minAge:     24 * time.Hour, // Default: don't delete objects less than 24h old
		keepRecent: 3,              // Keep last 3 generations
	}
}

// SetDryRun sets whether to perform a dry run
func (gc *GarbageCollector) SetDryRun(dryRun bool) {
	gc.dryRun = dryRun
}

// SetMinAge sets the minimum age for objects to be collected
func (gc *GarbageCollector) SetMinAge(age time.Duration) {
	gc.minAge = age
}

// SetKeepRecent sets how many recent generations to keep
func (gc *GarbageCollector) SetKeepRecent(count int) {
	gc.keepRecent = count
}

// OnProgress sets the progress callback
func (gc *GarbageCollector) OnProgress(callback func(GCProgress)) {
	gc.onProgress = callback
}

// OnComplete sets the completion callback
func (gc *GarbageCollector) OnComplete(callback func(GCResult)) {
	gc.onComplete = callback
}

// Run performs garbage collection
func (gc *GarbageCollector) Run() (*GCResult, error) {
	gc.mu.Lock()
	if gc.running {
		gc.mu.Unlock()
		return nil, fmt.Errorf("garbage collection already running")
	}
	gc.running = true
	gc.mu.Unlock()

	defer func() {
		gc.mu.Lock()
		gc.running = false
		gc.mu.Unlock()
	}()

	result := &GCResult{
		StartTime: time.Now(),
		DryRun:    gc.dryRun,
	}

	// Phase 1: Mark - Find all reachable objects
	gc.reportProgress(GCProgress{Phase: "marking"})
	reachable, err := gc.mark()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("mark phase failed: %v", err))
		return result, err
	}

	// Phase 2: Sweep - Delete unreachable objects
	gc.reportProgress(GCProgress{Phase: "sweeping", ObjectsMarked: int64(len(reachable))})
	deleted, bytesFreed, errors := gc.sweep(reachable)
	
	result.ObjectsDeleted = deleted
	result.BytesFreed = bytesFreed
	result.Errors = append(result.Errors, errors...)
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Update store metadata
	if !gc.dryRun {
		gc.store.mu.Lock()
		gc.store.metadata.LastGC = time.Now()
		gc.store.saveMetadata()
		gc.store.mu.Unlock()
	}

	// Report completion
	if gc.onComplete != nil {
		gc.onComplete(*result)
	}

	return result, nil
}

// mark finds all reachable objects
func (gc *GarbageCollector) mark() (map[string]bool, error) {
	reachable := make(map[string]bool)

	// Mark objects referenced by GC roots
	roots, err := gc.store.ListGCRoots()
	if err != nil {
		return nil, fmt.Errorf("failed to list GC roots: %w", err)
	}

	for _, hash := range roots {
		gc.markObject(hash, reachable)
	}

	// Mark objects referenced by profiles
	profiles, err := gc.listProfiles()
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}

	for _, profile := range profiles {
		for _, pkgHash := range profile.Packages {
			gc.markPackage(pkgHash, reachable)
		}
	}

	// Mark objects referenced by packages
	packages, err := gc.store.ListPackages()
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	for _, pkg := range packages {
		gc.markPackage(pkg.Hash, reachable)
	}

	return reachable, nil
}

// markObject marks an object and its dependencies as reachable
func (gc *GarbageCollector) markObject(hash string, reachable map[string]bool) {
	if reachable[hash] {
		return // Already marked
	}
	reachable[hash] = true

	// If this is a tree object, mark its children
	// (In a real implementation, we'd parse the tree and mark children)
}

// markPackage marks a package and its contents as reachable
func (gc *GarbageCollector) markPackage(hash string, reachable map[string]bool) {
	reachable[hash] = true

	// Mark package dependencies
	pkg, err := gc.store.GetPackage(hash)
	if err != nil {
		return
	}

	for _, dep := range pkg.Dependencies {
		gc.markPackage(dep, reachable)
	}
}

// listProfiles returns all profiles
func (gc *GarbageCollector) listProfiles() ([]*ProfileInfo, error) {
	entries, err := os.ReadDir(gc.store.profilesPath)
	if err != nil {
		return nil, err
	}

	var profiles []*ProfileInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		profile, err := gc.store.GetProfile(entry.Name())
		if err != nil {
			continue
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// sweep deletes unreachable objects
func (gc *GarbageCollector) sweep(reachable map[string]bool) (int64, int64, []string) {
	var deleted int64
	var bytesFreed int64
	var errors []string

	now := time.Now()

	// Walk through all objects
	err := filepath.Walk(gc.store.objectsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}

		hash := filepath.Base(path)

		// Check if reachable
		if reachable[hash] {
			return nil
		}

		// Check minimum age
		if now.Sub(info.ModTime()) < gc.minAge {
			return nil
		}

		// Delete object
		if !gc.dryRun {
			if err := os.Remove(path); err != nil {
				errors = append(errors, fmt.Sprintf("failed to delete %s: %v", path, err))
				return nil
			}
		}

		deleted++
		bytesFreed += info.Size()

		gc.reportProgress(GCProgress{
			Phase:          "sweeping",
			ObjectsDeleted: deleted,
			BytesFreed:     bytesFreed,
		})

		return nil
	})

	if err != nil {
		errors = append(errors, fmt.Sprintf("sweep walk error: %v", err))
	}

	// Clean up empty prefix directories
	if !gc.dryRun {
		gc.cleanupEmptyDirs()
	}

	return deleted, bytesFreed, errors
}

// cleanupEmptyDirs removes empty prefix directories
func (gc *GarbageCollector) cleanupEmptyDirs() {
	entries, err := os.ReadDir(gc.store.objectsPath)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(gc.store.objectsPath, entry.Name())
		subEntries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}

		if len(subEntries) == 0 {
			os.Remove(dirPath)
		}
	}
}

// reportProgress reports GC progress
func (gc *GarbageCollector) reportProgress(progress GCProgress) {
	if gc.onProgress != nil {
		gc.onProgress(progress)
	}
}

// IsRunning returns whether GC is currently running
func (gc *GarbageCollector) IsRunning() bool {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	return gc.running
}

// AutoGC performs automatic garbage collection based on thresholds
type AutoGC struct {
	gc           *GarbageCollector
	interval     time.Duration
	sizeThreshold int64 // Trigger GC when store exceeds this size
	
	running      bool
	done         chan struct{}
	mu           sync.Mutex
}

// NewAutoGC creates a new automatic garbage collector
func NewAutoGC(gc *GarbageCollector, interval time.Duration) *AutoGC {
	return &AutoGC{
		gc:            gc,
		interval:      interval,
		sizeThreshold: 10 * 1024 * 1024 * 1024, // 10GB default
		done:          make(chan struct{}),
	}
}

// SetSizeThreshold sets the size threshold for triggering GC
func (agc *AutoGC) SetSizeThreshold(size int64) {
	agc.sizeThreshold = size
}

// Start starts automatic garbage collection
func (agc *AutoGC) Start() {
	agc.mu.Lock()
	if agc.running {
		agc.mu.Unlock()
		return
	}
	agc.running = true
	agc.mu.Unlock()

	go agc.loop()
}

// Stop stops automatic garbage collection
func (agc *AutoGC) Stop() {
	agc.mu.Lock()
	if !agc.running {
		agc.mu.Unlock()
		return
	}
	agc.running = false
	agc.mu.Unlock()

	close(agc.done)
}

// loop runs the automatic GC loop
func (agc *AutoGC) loop() {
	ticker := time.NewTicker(agc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-agc.done:
			return
		case <-ticker.C:
			agc.checkAndRun()
		}
	}
}

// checkAndRun checks if GC should run and runs it
func (agc *AutoGC) checkAndRun() {
	stats := agc.gc.store.Stats()

	// Check if we should run GC
	if stats.TotalSize < agc.sizeThreshold {
		return
	}

	// Run GC
	agc.gc.Run()
}

// Optimize performs store optimization
func (s *Store) Optimize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Recompute metadata
	var objectCount int64
	var totalSize int64

	err := filepath.Walk(s.objectsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		objectCount++
		totalSize += info.Size()
		return nil
	})

	if err != nil {
		return err
	}

	s.metadata.ObjectCount = objectCount
	s.metadata.TotalSize = totalSize

	return s.saveMetadata()
}

// Compact compacts the store by removing fragmentation
func (s *Store) Compact() error {
	// In a real implementation, this would:
	// 1. Rewrite objects to remove fragmentation
	// 2. Rebuild indexes
	// 3. Optimize storage layout
	
	// For now, just optimize metadata
	return s.Optimize()
}
