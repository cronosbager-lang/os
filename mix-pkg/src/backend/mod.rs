// Package manager backends
pub mod alpm;
pub mod dpkg;
pub mod native;

use anyhow::Result;
use std::path::Path;

use crate::core::package::Package;

/// Backend trait for package management operations
pub trait Backend: Send + Sync {
    /// Get backend name
    fn name(&self) -> &str;
    
    /// Initialize the backend
    fn init(&mut self) -> Result<()>;
    
    /// Install a package
    fn install(&self, package: &Package) -> Result<()>;
    
    /// Remove a package
    fn remove(&self, name: &str) -> Result<()>;
    
    /// Check if package is installed
    fn is_installed(&self, name: &str) -> Result<bool>;
    
    /// Get installed package info
    fn get_installed(&self, name: &str) -> Result<Option<Package>>;
    
    /// List all installed packages
    fn list_installed(&self) -> Result<Vec<Package>>;
    
    /// Search for packages
    fn search(&self, query: &str) -> Result<Vec<Package>>;
    
    /// Get package from repository
    fn get_package(&self, name: &str) -> Result<Option<Package>>;
    
    /// Sync repository databases
    fn sync(&self) -> Result<()>;
    
    /// Upgrade all packages
    fn upgrade(&self) -> Result<Vec<Package>>;
    
    /// Get package dependencies
    fn get_dependencies(&self, name: &str) -> Result<Vec<String>>;
    
    /// Verify package integrity
    fn verify(&self, package: &Package) -> Result<bool>;
}

/// Backend type enum
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum BackendType {
    Alpm,   // Arch Linux Package Manager
    Dpkg,   // Debian Package Manager
    Native, // MIXOS Native
}

impl BackendType {
    pub fn from_str(s: &str) -> Option<Self> {
        match s.to_lowercase().as_str() {
            "alpm" | "pacman" | "arch" => Some(Self::Alpm),
            "dpkg" | "apt" | "debian" => Some(Self::Dpkg),
            "native" | "mixos" => Some(Self::Native),
            _ => None,
        }
    }
}

/// Create a backend instance
pub fn create_backend(backend_type: BackendType, root: &Path) -> Result<Box<dyn Backend>> {
    match backend_type {
        BackendType::Alpm => Ok(Box::new(alpm::AlpmBackend::new(root)?)),
        BackendType::Dpkg => Ok(Box::new(dpkg::DpkgBackend::new(root)?)),
        BackendType::Native => Ok(Box::new(native::NativeBackend::new(root)?)),
    }
}

/// Detect the best backend for the current system
pub fn detect_backend() -> BackendType {
    // Check for pacman
    if Path::new("/usr/bin/pacman").exists() {
        return BackendType::Alpm;
    }
    
    // Check for dpkg
    if Path::new("/usr/bin/dpkg").exists() {
        return BackendType::Dpkg;
    }
    
    // Default to native
    BackendType::Native
}
