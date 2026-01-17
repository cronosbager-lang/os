package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store provides content-addressable storage for MixOS
type Store struct {
	basePath string
	mu       sync.RWMutex
	
	// Paths
	objectsPath  string
	packagesPath string
	profilesPath string
	sourcesPath  string
	gcRootsPath  string
	
	// Metadata
	metadata *StoreMetadata
}

// StoreMetadata contains store metadata
type StoreMetadata struct {
	Version     string    `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	LastGC      time.Time `json:"last_gc,omitempty"`
	ObjectCount int64     `json:"object_count"`
	TotalSize   int64     `json:"total_size"`
}

// ObjectInfo contains information about a stored object
type ObjectInfo struct {
	Hash      string    `json:"hash"`
	Size      int64     `json:"size"`
	Type      string    `json:"type"` // "blob", "tree", "package"
	CreatedAt time.Time `json:"created_at"`
	RefCount  int       `json:"ref_count"`
}

// PackageInfo contains information about an installed package
type PackageInfo struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Hash         string            `json:"hash"`
	Dependencies []string          `json:"dependencies,omitempty"`
	InstalledAt  time.Time         `json:"installed_at"`
	Size         int64             `json:"size"`
	Files        []string          `json:"files,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ProfileInfo contains information about a profile
type ProfileInfo struct {
	Name       string    `json:"name"`
	Packages   []string  `json:"packages"` // Package hashes
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Generation int       `json:"generation"`
}

// NewStore creates a new store instance
func NewStore(basePath string) (*Store, error) {
	s := &Store{
		basePath:     basePath,
		objectsPath:  filepath.Join(basePath, "objects"),
		packagesPath: filepath.Join(basePath, "packages"),
		profilesPath: filepath.Join(basePath, "profiles"),
		sourcesPath:  filepath.Join(basePath, "sources"),
		gcRootsPath:  filepath.Join(basePath, "gc-roots"),
	}

	// Create directories
	dirs := []string{
		s.objectsPath,
		s.packagesPath,
		s.profilesPath,
		s.sourcesPath,
		s.gcRootsPath,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Load or create metadata
	if err := s.loadMetadata(); err != nil {
		s.metadata = &StoreMetadata{
			Version:   "1.0",
			CreatedAt: time.Now(),
		}
		s.saveMetadata()
	}

	return s, nil
}

// loadMetadata loads store metadata
func (s *Store) loadMetadata() error {
	path := filepath.Join(s.basePath, "metadata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	s.metadata = &StoreMetadata{}
	return json.Unmarshal(data, s.metadata)
}

// saveMetadata saves store metadata
func (s *Store) saveMetadata() error {
	path := filepath.Join(s.basePath, "metadata.json")
	data, err := json.MarshalIndent(s.metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// HashContent calculates the SHA256 hash of content
func HashContent(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// HashFile calculates the SHA256 hash of a file
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// objectPath returns the path for an object with the given hash
func (s *Store) objectPath(hash string) string {
	// Use first 2 characters as prefix directory
	prefix := hash[:2]
	return filepath.Join(s.objectsPath, prefix, hash)
}

// AddObject adds content to the store and returns its hash
func (s *Store) AddObject(content []byte) (string, error) {
	hash := HashContent(content)
	return hash, s.AddObjectWithHash(hash, content)
}

// AddObjectWithHash adds content with a pre-calculated hash
func (s *Store) AddObjectWithHash(hash string, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.objectPath(hash)

	// Check if already exists
	if _, err := os.Stat(path); err == nil {
		return nil // Already exists
	}

	// Create prefix directory
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create object directory: %w", err)
	}

	// Write content
	if err := os.WriteFile(path, content, 0444); err != nil {
		return fmt.Errorf("failed to write object: %w", err)
	}

	// Update metadata
	s.metadata.ObjectCount++
	s.metadata.TotalSize += int64(len(content))
	s.saveMetadata()

	return nil
}

// AddObjectFromFile adds a file to the store
func (s *Store) AddObjectFromFile(srcPath string) (string, error) {
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return s.AddObject(content)
}

// GetObject retrieves an object by hash
func (s *Store) GetObject(hash string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.objectPath(hash)
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("object not found: %s", hash)
		}
		return nil, err
	}

	return content, nil
}

// HasObject checks if an object exists
func (s *Store) HasObject(hash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.objectPath(hash)
	_, err := os.Stat(path)
	return err == nil
}

// DeleteObject removes an object from the store
func (s *Store) DeleteObject(hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.objectPath(hash)
	
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if err := os.Remove(path); err != nil {
		return err
	}

	// Update metadata
	s.metadata.ObjectCount--
	s.metadata.TotalSize -= info.Size()
	s.saveMetadata()

	return nil
}

// GetObjectInfo returns information about an object
func (s *Store) GetObjectInfo(hash string) (*ObjectInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.objectPath(hash)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("object not found: %s", hash)
		}
		return nil, err
	}

	return &ObjectInfo{
		Hash:      hash,
		Size:      info.Size(),
		Type:      "blob",
		CreatedAt: info.ModTime(),
	}, nil
}

// packagePath returns the path for a package
func (s *Store) packagePath(hash string) string {
	return filepath.Join(s.packagesPath, hash)
}

// InstallPackage installs a package to the store
func (s *Store) InstallPackage(pkg *PackageInfo, content []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate hash if not provided
	if pkg.Hash == "" {
		pkg.Hash = HashContent(content)
	}

	pkgPath := s.packagePath(pkg.Hash)

	// Create package directory
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Write package content (assuming it's a tarball or similar)
	contentPath := filepath.Join(pkgPath, "content")
	if err := os.WriteFile(contentPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write package content: %w", err)
	}

	// Write package info
	pkg.InstalledAt = time.Now()
	pkg.Size = int64(len(content))
	
	infoPath := filepath.Join(pkgPath, "info.json")
	infoData, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(infoPath, infoData, 0644); err != nil {
		return fmt.Errorf("failed to write package info: %w", err)
	}

	return nil
}

// GetPackage retrieves package information
func (s *Store) GetPackage(hash string) (*PackageInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	infoPath := filepath.Join(s.packagePath(hash), "info.json")
	data, err := os.ReadFile(infoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("package not found: %s", hash)
		}
		return nil, err
	}

	var pkg PackageInfo
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	return &pkg, nil
}

// ListPackages returns all installed packages
func (s *Store) ListPackages() ([]*PackageInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.packagesPath)
	if err != nil {
		return nil, err
	}

	var packages []*PackageInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pkg, err := s.GetPackage(entry.Name())
		if err != nil {
			continue
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

// RemovePackage removes a package from the store
func (s *Store) RemovePackage(hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pkgPath := s.packagePath(hash)
	return os.RemoveAll(pkgPath)
}

// profilePath returns the path for a profile
func (s *Store) profilePath(name string) string {
	return filepath.Join(s.profilesPath, name)
}

// CreateProfile creates a new profile
func (s *Store) CreateProfile(name string) (*ProfileInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profilePath := s.profilePath(name)
	
	// Check if exists
	if _, err := os.Stat(profilePath); err == nil {
		return nil, fmt.Errorf("profile already exists: %s", name)
	}

	// Create profile directory
	if err := os.MkdirAll(profilePath, 0755); err != nil {
		return nil, err
	}

	profile := &ProfileInfo{
		Name:       name,
		Packages:   []string{},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Generation: 1,
	}

	// Save profile info
	infoPath := filepath.Join(profilePath, "info.json")
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(infoPath, data, 0644); err != nil {
		return nil, err
	}

	return profile, nil
}

// GetProfile retrieves a profile
func (s *Store) GetProfile(name string) (*ProfileInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	infoPath := filepath.Join(s.profilePath(name), "info.json")
	data, err := os.ReadFile(infoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("profile not found: %s", name)
		}
		return nil, err
	}

	var profile ProfileInfo
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

// UpdateProfile updates a profile with new packages
func (s *Store) UpdateProfile(name string, packages []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, err := s.GetProfile(name)
	if err != nil {
		return err
	}

	profile.Packages = packages
	profile.UpdatedAt = time.Now()
	profile.Generation++

	infoPath := filepath.Join(s.profilePath(name), "info.json")
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(infoPath, data, 0644)
}

// LinkPackageToProfile links a package to a profile
func (s *Store) LinkPackageToProfile(profileName, packageHash string) error {
	profile, err := s.GetProfile(profileName)
	if err != nil {
		return err
	}

	// Check if already linked
	for _, pkg := range profile.Packages {
		if pkg == packageHash {
			return nil
		}
	}

	profile.Packages = append(profile.Packages, packageHash)
	return s.UpdateProfile(profileName, profile.Packages)
}

// UnlinkPackageFromProfile unlinks a package from a profile
func (s *Store) UnlinkPackageFromProfile(profileName, packageHash string) error {
	profile, err := s.GetProfile(profileName)
	if err != nil {
		return err
	}

	var newPackages []string
	for _, pkg := range profile.Packages {
		if pkg != packageHash {
			newPackages = append(newPackages, pkg)
		}
	}

	return s.UpdateProfile(profileName, newPackages)
}

// AddGCRoot adds a GC root to protect an object from garbage collection
func (s *Store) AddGCRoot(name, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rootPath := filepath.Join(s.gcRootsPath, name)
	return os.WriteFile(rootPath, []byte(hash), 0644)
}

// RemoveGCRoot removes a GC root
func (s *Store) RemoveGCRoot(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rootPath := filepath.Join(s.gcRootsPath, name)
	return os.Remove(rootPath)
}

// ListGCRoots returns all GC roots
func (s *Store) ListGCRoots() (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.gcRootsPath)
	if err != nil {
		return nil, err
	}

	roots := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := os.ReadFile(filepath.Join(s.gcRootsPath, entry.Name()))
		if err != nil {
			continue
		}
		roots[entry.Name()] = string(data)
	}

	return roots, nil
}

// Stats returns store statistics
func (s *Store) Stats() *StoreStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &StoreStats{
		BasePath:    s.basePath,
		ObjectCount: s.metadata.ObjectCount,
		TotalSize:   s.metadata.TotalSize,
		CreatedAt:   s.metadata.CreatedAt,
		LastGC:      s.metadata.LastGC,
	}

	// Count packages
	if entries, err := os.ReadDir(s.packagesPath); err == nil {
		stats.PackageCount = int64(len(entries))
	}

	// Count profiles
	if entries, err := os.ReadDir(s.profilesPath); err == nil {
		stats.ProfileCount = int64(len(entries))
	}

	// Count GC roots
	if entries, err := os.ReadDir(s.gcRootsPath); err == nil {
		stats.GCRootCount = int64(len(entries))
	}

	return stats
}

// StoreStats contains store statistics
type StoreStats struct {
	BasePath     string    `json:"base_path"`
	ObjectCount  int64     `json:"object_count"`
	PackageCount int64     `json:"package_count"`
	ProfileCount int64     `json:"profile_count"`
	GCRootCount  int64     `json:"gc_root_count"`
	TotalSize    int64     `json:"total_size"`
	CreatedAt    time.Time `json:"created_at"`
	LastGC       time.Time `json:"last_gc,omitempty"`
}

// Verify verifies the integrity of the store
func (s *Store) Verify() (*VerifyResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := &VerifyResult{
		StartTime: time.Now(),
	}

	// Verify objects
	err := filepath.Walk(s.objectsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		result.ObjectsChecked++

		// Verify hash
		content, err := os.ReadFile(path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to read %s: %v", path, err))
			return nil
		}

		expectedHash := filepath.Base(path)
		actualHash := HashContent(content)

		if expectedHash != actualHash {
			result.Corrupted++
			result.Errors = append(result.Errors, fmt.Sprintf("hash mismatch for %s: expected %s, got %s", path, expectedHash, actualHash))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// VerifyResult contains the result of a store verification
type VerifyResult struct {
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time"`
	Duration       time.Duration `json:"duration"`
	ObjectsChecked int64         `json:"objects_checked"`
	Corrupted      int64         `json:"corrupted"`
	Errors         []string      `json:"errors,omitempty"`
}
