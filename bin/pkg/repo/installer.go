package repo

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"mixos.dev/init/pkg/store"
)

// Installer handles package installation
type Installer struct {
	store      *store.Store
	repoMgr    *RepositoryManager
	workDir    string
	
	// Installation state
	installing map[string]bool
	mu         sync.Mutex
	
	// Callbacks
	onProgress func(InstallProgress)
	onComplete func(InstallResult)
}

// InstallProgress contains installation progress information
type InstallProgress struct {
	Package     string  `json:"package"`
	Phase       string  `json:"phase"` // "download", "extract", "build", "install", "hooks"
	Progress    float64 `json:"progress"`
	Message     string  `json:"message"`
}

// InstallResult contains the result of an installation
type InstallResult struct {
	Package     string        `json:"package"`
	Version     string        `json:"version"`
	Success     bool          `json:"success"`
	Error       string        `json:"error,omitempty"`
	Hash        string        `json:"hash,omitempty"`
	Duration    time.Duration `json:"duration"`
	InstalledAt time.Time     `json:"installed_at"`
}

// InstallOptions contains options for package installation
type InstallOptions struct {
	Force           bool     // Force reinstall even if already installed
	NoDeps          bool     // Don't install dependencies
	BuildOnly       bool     // Only build, don't install
	KeepBuildDir    bool     // Keep build directory after installation
	Profile         string   // Profile to install to
	ExtraEnv        map[string]string // Extra environment variables
}

// NewInstaller creates a new package installer
func NewInstaller(s *store.Store, repoMgr *RepositoryManager, workDir string) *Installer {
	return &Installer{
		store:      s,
		repoMgr:    repoMgr,
		workDir:    workDir,
		installing: make(map[string]bool),
	}
}

// OnProgress sets the progress callback
func (i *Installer) OnProgress(callback func(InstallProgress)) {
	i.onProgress = callback
}

// OnComplete sets the completion callback
func (i *Installer) OnComplete(callback func(InstallResult)) {
	i.onComplete = callback
}

// Install installs a package and its dependencies
func (i *Installer) Install(name string, opts *InstallOptions) (*InstallResult, error) {
	if opts == nil {
		opts = &InstallOptions{}
	}
	
	startTime := time.Now()
	result := &InstallResult{
		Package: name,
	}
	
	// Check if already installing
	i.mu.Lock()
	if i.installing[name] {
		i.mu.Unlock()
		return nil, fmt.Errorf("package %s is already being installed", name)
	}
	i.installing[name] = true
	i.mu.Unlock()
	
	defer func() {
		i.mu.Lock()
		delete(i.installing, name)
		i.mu.Unlock()
	}()
	
	// Get package info
	pkg, repoName, err := i.repoMgr.GetPackage(name)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	result.Version = pkg.Version
	
	// Install dependencies first
	if !opts.NoDeps {
		for _, dep := range pkg.Dependencies {
			i.reportProgress(InstallProgress{
				Package: name,
				Phase:   "dependencies",
				Message: fmt.Sprintf("Installing dependency: %s", dep),
			})
			
			if _, err := i.Install(dep, opts); err != nil {
				result.Error = fmt.Sprintf("failed to install dependency %s: %v", dep, err)
				return result, fmt.Errorf(result.Error)
			}
		}
	}
	
	// Create build directory
	buildDir := filepath.Join(i.workDir, name+"-"+pkg.Version)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		result.Error = fmt.Sprintf("failed to create build directory: %v", err)
		return result, fmt.Errorf(result.Error)
	}
	
	if !opts.KeepBuildDir {
		defer os.RemoveAll(buildDir)
	}
	
	// Download package
	i.reportProgress(InstallProgress{
		Package: name,
		Phase:   "download",
		Message: "Downloading package...",
	})
	
	pkgPath := filepath.Join(buildDir, "package")
	if err := i.repoMgr.DownloadPackage(name, repoName, pkgPath); err != nil {
		result.Error = fmt.Sprintf("failed to download package: %v", err)
		return result, fmt.Errorf(result.Error)
	}
	
	// Load manifest
	repo, _ := i.repoMgr.GetRepository(repoName)
	var manifest *PackageManifest
	
	if repo.Type == "local" {
		manifestPath := filepath.Join(repo.URL, pkg.Filename, "package.toml")
		manifest, err = i.repoMgr.loadManifest(manifestPath)
		if err != nil {
			result.Error = fmt.Sprintf("failed to load manifest: %v", err)
			return result, fmt.Errorf(result.Error)
		}
	}
	
	// Build package
	i.reportProgress(InstallProgress{
		Package: name,
		Phase:   "build",
		Message: "Building package...",
	})
	
	outDir := filepath.Join(buildDir, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		result.Error = fmt.Sprintf("failed to create output directory: %v", err)
		return result, fmt.Errorf(result.Error)
	}
	
	if manifest != nil && manifest.Build.Type == "custom" {
		if err := i.runBuildScript(manifest, buildDir, outDir, opts.ExtraEnv); err != nil {
			result.Error = fmt.Sprintf("build failed: %v", err)
			return result, fmt.Errorf(result.Error)
		}
	}
	
	if opts.BuildOnly {
		result.Success = true
		result.Duration = time.Since(startTime)
		return result, nil
	}
	
	// Run pre-install hook
	if manifest != nil && manifest.Hooks.PreInstall != "" {
		i.reportProgress(InstallProgress{
			Package: name,
			Phase:   "hooks",
			Message: "Running pre-install hook...",
		})
		
		if err := i.runHook(manifest.Hooks.PreInstall, outDir); err != nil {
			result.Error = fmt.Sprintf("pre-install hook failed: %v", err)
			return result, fmt.Errorf(result.Error)
		}
	}
	
	// Install to store
	i.reportProgress(InstallProgress{
		Package: name,
		Phase:   "install",
		Message: "Installing to store...",
	})
	
	// Read output directory contents
	content, err := i.packDirectory(outDir)
	if err != nil {
		result.Error = fmt.Sprintf("failed to pack output: %v", err)
		return result, fmt.Errorf(result.Error)
	}
	
	// Create package info for store
	storePackage := &store.PackageInfo{
		Name:         name,
		Version:      pkg.Version,
		Dependencies: pkg.Dependencies,
		Metadata: map[string]string{
			"description": pkg.Description,
			"license":     pkg.License,
			"repository":  repoName,
		},
	}
	
	if err := i.store.InstallPackage(storePackage, content); err != nil {
		result.Error = fmt.Sprintf("failed to install to store: %v", err)
		return result, fmt.Errorf(result.Error)
	}
	
	result.Hash = storePackage.Hash
	
	// Link to profile
	profile := opts.Profile
	if profile == "" {
		profile = "default"
	}
	
	// Ensure profile exists
	if _, err := i.store.GetProfile(profile); err != nil {
		if _, err := i.store.CreateProfile(profile); err != nil {
			result.Error = fmt.Sprintf("failed to create profile: %v", err)
			return result, fmt.Errorf(result.Error)
		}
	}
	
	if err := i.store.LinkPackageToProfile(profile, storePackage.Hash); err != nil {
		result.Error = fmt.Sprintf("failed to link to profile: %v", err)
		return result, fmt.Errorf(result.Error)
	}
	
	// Run post-install hook
	if manifest != nil && manifest.Hooks.PostInstall != "" {
		i.reportProgress(InstallProgress{
			Package: name,
			Phase:   "hooks",
			Message: "Running post-install hook...",
		})
		
		if err := i.runHook(manifest.Hooks.PostInstall, outDir); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: post-install hook failed: %v\n", err)
		}
	}
	
	result.Success = true
	result.Duration = time.Since(startTime)
	result.InstalledAt = time.Now()
	
	// Report completion
	if i.onComplete != nil {
		i.onComplete(*result)
	}
	
	return result, nil
}

// runBuildScript runs the build script
func (i *Installer) runBuildScript(manifest *PackageManifest, buildDir, outDir string, extraEnv map[string]string) error {
	// Create build script
	buildScript := filepath.Join(buildDir, "build.sh")
	if err := os.WriteFile(buildScript, []byte(manifest.Build.Custom.Build), 0755); err != nil {
		return err
	}
	
	// Set up environment
	env := os.Environ()
	env = append(env, fmt.Sprintf("out=%s", outDir))
	
	for k, v := range manifest.Build.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range extraEnv {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	
	// Deterministic environment
	env = append(env, "SOURCE_DATE_EPOCH=1")
	env = append(env, "TZ=UTC")
	env = append(env, "LC_ALL=C")
	
	// Run build
	cmd := exec.Command("/bin/sh", buildScript)
	cmd.Dir = buildDir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build script failed: %w", err)
	}
	
	// Run install script if present
	if manifest.Build.Custom.Install != "" {
		installScript := filepath.Join(buildDir, "install.sh")
		if err := os.WriteFile(installScript, []byte(manifest.Build.Custom.Install), 0755); err != nil {
			return err
		}
		
		cmd = exec.Command("/bin/sh", installScript)
		cmd.Dir = buildDir
		cmd.Env = env
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("install script failed: %w", err)
		}
	}
	
	return nil
}

// runHook runs a hook script
func (i *Installer) runHook(script, workDir string) error {
	hookScript := filepath.Join(workDir, "hook.sh")
	if err := os.WriteFile(hookScript, []byte(script), 0755); err != nil {
		return err
	}
	defer os.Remove(hookScript)
	
	cmd := exec.Command("/bin/sh", hookScript)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

// packDirectory packs a directory into a byte slice
func (i *Installer) packDirectory(dir string) ([]byte, error) {
	// For simplicity, we'll create a JSON manifest of the directory
	// In a real implementation, this would create a tarball or similar
	
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(dir, path)
			files = append(files, relPath)
		}
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	manifest := map[string]interface{}{
		"files":     files,
		"timestamp": time.Now(),
	}
	
	return json.Marshal(manifest)
}

// reportProgress reports installation progress
func (i *Installer) reportProgress(progress InstallProgress) {
	if i.onProgress != nil {
		i.onProgress(progress)
	}
}

// Remove removes a package
func (i *Installer) Remove(name string, profile string) error {
	if profile == "" {
		profile = "default"
	}
	
	// Find package in profile
	profileInfo, err := i.store.GetProfile(profile)
	if err != nil {
		return fmt.Errorf("profile not found: %s", profile)
	}
	
	// Find package hash
	var pkgHash string
	for _, hash := range profileInfo.Packages {
		pkg, err := i.store.GetPackage(hash)
		if err != nil {
			continue
		}
		if pkg.Name == name {
			pkgHash = hash
			break
		}
	}
	
	if pkgHash == "" {
		return fmt.Errorf("package not found in profile: %s", name)
	}
	
	// Unlink from profile
	if err := i.store.UnlinkPackageFromProfile(profile, pkgHash); err != nil {
		return err
	}
	
	return nil
}

// ListInstalled lists installed packages in a profile
func (i *Installer) ListInstalled(profile string) ([]*store.PackageInfo, error) {
	if profile == "" {
		profile = "default"
	}
	
	profileInfo, err := i.store.GetProfile(profile)
	if err != nil {
		return nil, err
	}
	
	var packages []*store.PackageInfo
	for _, hash := range profileInfo.Packages {
		pkg, err := i.store.GetPackage(hash)
		if err != nil {
			continue
		}
		packages = append(packages, pkg)
	}
	
	return packages, nil
}

// Upgrade upgrades a package to the latest version
func (i *Installer) Upgrade(name string, opts *InstallOptions) (*InstallResult, error) {
	if opts == nil {
		opts = &InstallOptions{}
	}
	opts.Force = true
	
	return i.Install(name, opts)
}

// UpgradeAll upgrades all packages in a profile
func (i *Installer) UpgradeAll(profile string, opts *InstallOptions) ([]*InstallResult, error) {
	packages, err := i.ListInstalled(profile)
	if err != nil {
		return nil, err
	}
	
	var results []*InstallResult
	for _, pkg := range packages {
		result, err := i.Upgrade(pkg.Name, opts)
		if err != nil {
			result = &InstallResult{
				Package: pkg.Name,
				Error:   err.Error(),
			}
		}
		results = append(results, result)
	}
	
	return results, nil
}

// Transaction represents an atomic installation transaction
type Transaction struct {
	installer *Installer
	packages  []string
	opts      *InstallOptions
	results   []*InstallResult
	committed bool
}

// NewTransaction creates a new installation transaction
func (i *Installer) NewTransaction(packages []string, opts *InstallOptions) *Transaction {
	return &Transaction{
		installer: i,
		packages:  packages,
		opts:      opts,
	}
}

// Execute executes the transaction
func (t *Transaction) Execute() error {
	for _, pkg := range t.packages {
		result, err := t.installer.Install(pkg, t.opts)
		t.results = append(t.results, result)
		
		if err != nil {
			// Rollback on failure
			t.Rollback()
			return err
		}
	}
	
	t.committed = true
	return nil
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback() {
	if t.committed {
		return
	}
	
	// Remove installed packages in reverse order
	for i := len(t.results) - 1; i >= 0; i-- {
		result := t.results[i]
		if result.Success {
			t.installer.Remove(result.Package, t.opts.Profile)
		}
	}
}

// Results returns the transaction results
func (t *Transaction) Results() []*InstallResult {
	return t.results
}
