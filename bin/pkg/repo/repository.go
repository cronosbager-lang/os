package repo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

// Repository represents a package repository
type Repository struct {
	Name     string `json:"name" toml:"name"`
	URL      string `json:"url" toml:"url"`
	Type     string `json:"type" toml:"type"` // "local" or "remote"
	Priority int    `json:"priority" toml:"priority"`
	Enabled  bool   `json:"enabled" toml:"enabled"`
	GPGKey   string `json:"gpg_key,omitempty" toml:"gpg_key"`
	
	// Runtime state
	index    *RepositoryIndex
	lastSync time.Time
	mu       sync.RWMutex
}

// RepositoryIndex contains the package index for a repository
type RepositoryIndex struct {
	Version   string                    `json:"version"`
	Timestamp time.Time                 `json:"timestamp"`
	Packages  map[string]*PackageEntry  `json:"packages"`
}

// PackageEntry represents a package in the repository index
type PackageEntry struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	License      string            `json:"license,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
	Size         int64             `json:"size"`
	SHA256       string            `json:"sha256"`
	Filename     string            `json:"filename"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// PackageManifest represents a package.toml manifest
type PackageManifest struct {
	Package      PackageInfo       `toml:"package"`
	Source       SourceInfo        `toml:"source"`
	Dependencies DependencyInfo    `toml:"dependencies"`
	Build        BuildInfo         `toml:"build"`
	Install      InstallInfo       `toml:"install"`
	Hooks        HooksInfo         `toml:"hooks"`
}

type PackageInfo struct {
	Name        string   `toml:"name"`
	Version     string   `toml:"version"`
	Description string   `toml:"description"`
	License     string   `toml:"license"`
	Homepage    string   `toml:"homepage"`
	Maintainers []string `toml:"maintainers"`
	Keywords    []string `toml:"keywords"`
}

type SourceInfo struct {
	URL    string `toml:"url"`
	SHA256 string `toml:"sha256"`
	Type   string `toml:"type"`
}

type DependencyInfo struct {
	Runtime  map[string]string `toml:"runtime"`
	Build    map[string]string `toml:"build"`
	Optional map[string]string `toml:"optional"`
}

type BuildInfo struct {
	Type   string            `toml:"type"`
	Env    map[string]string `toml:"env"`
	Custom CustomBuildInfo   `toml:"custom"`
}

type CustomBuildInfo struct {
	Configure string `toml:"configure"`
	Build     string `toml:"build"`
	Install   string `toml:"install"`
}

type InstallInfo struct {
	Prefix string `toml:"prefix"`
}

type HooksInfo struct {
	PreInstall  string `toml:"pre_install"`
	PostInstall string `toml:"post_install"`
	PreRemove   string `toml:"pre_remove"`
	PostRemove  string `toml:"post_remove"`
}

// RepositoryManager manages multiple repositories
type RepositoryManager struct {
	repos     []*Repository
	cachePath string
	mu        sync.RWMutex
	
	// HTTP client for remote repos
	client *http.Client
}

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager(cachePath string) *RepositoryManager {
	return &RepositoryManager{
		repos:     make([]*Repository, 0),
		cachePath: cachePath,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AddRepository adds a repository
func (rm *RepositoryManager) AddRepository(repo *Repository) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.repos = append(rm.repos, repo)
	rm.sortRepos()
}

// RemoveRepository removes a repository by name
func (rm *RepositoryManager) RemoveRepository(name string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	for i, repo := range rm.repos {
		if repo.Name == name {
			rm.repos = append(rm.repos[:i], rm.repos[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("repository not found: %s", name)
}

// GetRepository returns a repository by name
func (rm *RepositoryManager) GetRepository(name string) (*Repository, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	for _, repo := range rm.repos {
		if repo.Name == name {
			return repo, true
		}
	}
	return nil, false
}

// ListRepositories returns all repositories
func (rm *RepositoryManager) ListRepositories() []*Repository {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	repos := make([]*Repository, len(rm.repos))
	copy(repos, rm.repos)
	return repos
}

// sortRepos sorts repositories by priority
func (rm *RepositoryManager) sortRepos() {
	sort.Slice(rm.repos, func(i, j int) bool {
		return rm.repos[i].Priority < rm.repos[j].Priority
	})
}

// SyncAll syncs all enabled repositories
func (rm *RepositoryManager) SyncAll() error {
	rm.mu.RLock()
	repos := make([]*Repository, len(rm.repos))
	copy(repos, rm.repos)
	rm.mu.RUnlock()
	
	var errs []error
	for _, repo := range repos {
		if !repo.Enabled {
			continue
		}
		if err := rm.SyncRepository(repo.Name); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", repo.Name, err))
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("sync errors: %v", errs)
	}
	return nil
}

// SyncRepository syncs a specific repository
func (rm *RepositoryManager) SyncRepository(name string) error {
	repo, ok := rm.GetRepository(name)
	if !ok {
		return fmt.Errorf("repository not found: %s", name)
	}
	
	repo.mu.Lock()
	defer repo.mu.Unlock()
	
	var index *RepositoryIndex
	var err error
	
	if repo.Type == "local" {
		index, err = rm.syncLocalRepo(repo)
	} else {
		index, err = rm.syncRemoteRepo(repo)
	}
	
	if err != nil {
		return err
	}
	
	repo.index = index
	repo.lastSync = time.Now()
	
	// Cache the index
	return rm.cacheIndex(repo.Name, index)
}

// syncLocalRepo syncs a local repository
func (rm *RepositoryManager) syncLocalRepo(repo *Repository) (*RepositoryIndex, error) {
	indexPath := filepath.Join(repo.URL, "index.json")
	
	data, err := os.ReadFile(indexPath)
	if err != nil {
		// Try to build index from packages
		return rm.buildLocalIndex(repo.URL)
	}
	
	var index RepositoryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}
	
	return &index, nil
}

// buildLocalIndex builds an index from local packages
func (rm *RepositoryManager) buildLocalIndex(path string) (*RepositoryIndex, error) {
	index := &RepositoryIndex{
		Version:   "1.0",
		Timestamp: time.Now(),
		Packages:  make(map[string]*PackageEntry),
	}
	
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		manifestPath := filepath.Join(path, entry.Name(), "package.toml")
		manifest, err := rm.loadManifest(manifestPath)
		if err != nil {
			continue
		}
		
		index.Packages[manifest.Package.Name] = &PackageEntry{
			Name:        manifest.Package.Name,
			Version:     manifest.Package.Version,
			Description: manifest.Package.Description,
			License:     manifest.Package.License,
			Filename:    entry.Name(),
		}
	}
	
	return index, nil
}

// syncRemoteRepo syncs a remote repository
func (rm *RepositoryManager) syncRemoteRepo(repo *Repository) (*RepositoryIndex, error) {
	indexURL := repo.URL + "/index.json"
	
	resp, err := rm.client.Get(indexURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch index: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch index: HTTP %d", resp.StatusCode)
	}
	
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var index RepositoryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}
	
	return &index, nil
}

// loadManifest loads a package manifest
func (rm *RepositoryManager) loadManifest(path string) (*PackageManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var manifest PackageManifest
	if _, err := toml.Decode(string(data), &manifest); err != nil {
		return nil, err
	}
	
	return &manifest, nil
}

// cacheIndex caches a repository index
func (rm *RepositoryManager) cacheIndex(name string, index *RepositoryIndex) error {
	if rm.cachePath == "" {
		return nil
	}
	
	cachePath := filepath.Join(rm.cachePath, name+".json")
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return err
	}
	
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(cachePath, data, 0644)
}

// loadCachedIndex loads a cached repository index
func (rm *RepositoryManager) loadCachedIndex(name string) (*RepositoryIndex, error) {
	if rm.cachePath == "" {
		return nil, fmt.Errorf("no cache path configured")
	}
	
	cachePath := filepath.Join(rm.cachePath, name+".json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}
	
	var index RepositoryIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}
	
	return &index, nil
}

// SearchPackage searches for a package across all repositories
func (rm *RepositoryManager) SearchPackage(query string) []*SearchResult {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	var results []*SearchResult
	
	for _, repo := range rm.repos {
		if !repo.Enabled {
			continue
		}
		
		repo.mu.RLock()
		if repo.index == nil {
			repo.mu.RUnlock()
			continue
		}
		
		for name, pkg := range repo.index.Packages {
			if matchesQuery(name, pkg.Description, query) {
				results = append(results, &SearchResult{
					Package:    pkg,
					Repository: repo.Name,
				})
			}
		}
		repo.mu.RUnlock()
	}
	
	return results
}

// SearchResult represents a package search result
type SearchResult struct {
	Package    *PackageEntry
	Repository string
}

// matchesQuery checks if a package matches a search query
func matchesQuery(name, description, query string) bool {
	// Simple substring match
	return contains(name, query) || contains(description, query)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetPackage gets a package from the best available repository
func (rm *RepositoryManager) GetPackage(name string) (*PackageEntry, string, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	for _, repo := range rm.repos {
		if !repo.Enabled {
			continue
		}
		
		repo.mu.RLock()
		if repo.index == nil {
			repo.mu.RUnlock()
			continue
		}
		
		if pkg, ok := repo.index.Packages[name]; ok {
			repo.mu.RUnlock()
			return pkg, repo.Name, nil
		}
		repo.mu.RUnlock()
	}
	
	return nil, "", fmt.Errorf("package not found: %s", name)
}

// DownloadPackage downloads a package from a repository
func (rm *RepositoryManager) DownloadPackage(name, repoName, destPath string) error {
	repo, ok := rm.GetRepository(repoName)
	if !ok {
		return fmt.Errorf("repository not found: %s", repoName)
	}
	
	repo.mu.RLock()
	pkg, ok := repo.index.Packages[name]
	repo.mu.RUnlock()
	
	if !ok {
		return fmt.Errorf("package not found in repository: %s", name)
	}
	
	if repo.Type == "local" {
		// Copy from local repository
		srcPath := filepath.Join(repo.URL, pkg.Filename)
		return copyFile(srcPath, destPath)
	}
	
	// Download from remote repository
	pkgURL := repo.URL + "/" + pkg.Filename
	return rm.downloadFile(pkgURL, destPath, pkg.SHA256)
}

// downloadFile downloads a file and verifies its checksum
func (rm *RepositoryManager) downloadFile(url, destPath, expectedSHA256 string) error {
	resp, err := rm.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	
	// Create destination file
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	
	// Download and compute hash
	h := sha256.New()
	writer := io.MultiWriter(f, h)
	
	if _, err := io.Copy(writer, resp.Body); err != nil {
		os.Remove(destPath)
		return err
	}
	
	// Verify checksum
	actualSHA256 := hex.EncodeToString(h.Sum(nil))
	if expectedSHA256 != "" && actualSHA256 != expectedSHA256 {
		os.Remove(destPath)
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, actualSHA256)
	}
	
	return nil
}

// copyFile copies a file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	
	_, err = io.Copy(dstFile, srcFile)
	return err
}

// GenerateIndex generates an index for a local repository
func GenerateIndex(repoPath string) (*RepositoryIndex, error) {
	index := &RepositoryIndex{
		Version:   "1.0",
		Timestamp: time.Now(),
		Packages:  make(map[string]*PackageEntry),
	}
	
	entries, err := os.ReadDir(repoPath)
	if err != nil {
		return nil, err
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		pkgPath := filepath.Join(repoPath, entry.Name())
		manifestPath := filepath.Join(pkgPath, "package.toml")
		
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		
		var manifest PackageManifest
		if _, err := toml.Decode(string(data), &manifest); err != nil {
			continue
		}
		
		// Get package size
		var size int64
		filepath.Walk(pkgPath, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				size += info.Size()
			}
			return nil
		})
		
		// Compute dependencies list
		var deps []string
		for dep := range manifest.Dependencies.Runtime {
			deps = append(deps, dep)
		}
		
		index.Packages[manifest.Package.Name] = &PackageEntry{
			Name:         manifest.Package.Name,
			Version:      manifest.Package.Version,
			Description:  manifest.Package.Description,
			License:      manifest.Package.License,
			Dependencies: deps,
			Size:         size,
			Filename:     entry.Name(),
		}
	}
	
	return index, nil
}

// SaveIndex saves an index to a file
func SaveIndex(index *RepositoryIndex, path string) error {
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
