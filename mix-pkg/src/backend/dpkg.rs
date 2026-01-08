// DPKG (Debian Package Manager) Backend
use anyhow::{Result, Context, bail};
use std::path::{Path, PathBuf};
use std::process::Command;
use std::collections::HashMap;

use super::Backend;
use crate::core::package::Package;

/// DPKG Backend implementation
pub struct DpkgBackend {
    root: PathBuf,
    db_path: PathBuf,
    cache_path: PathBuf,
    status_file: PathBuf,
}

impl DpkgBackend {
    pub fn new(root: &Path) -> Result<Self> {
        let root = root.to_path_buf();
        let db_path = root.join("var/lib/dpkg");
        let cache_path = root.join("var/cache/apt/archives");
        let status_file = db_path.join("status");
        
        Ok(Self {
            root,
            db_path,
            cache_path,
            status_file,
        })
    }
    
    fn run_dpkg(&self, args: &[&str]) -> Result<String> {
        let output = Command::new("dpkg")
            .args(args)
            .arg("--root")
            .arg(&self.root)
            .arg("--admindir")
            .arg(&self.db_path)
            .output()
            .context("Failed to execute dpkg")?;
        
        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr);
            bail!("dpkg failed: {}", stderr);
        }
        
        Ok(String::from_utf8_lossy(&output.stdout).to_string())
    }
    
    fn run_apt(&self, args: &[&str]) -> Result<String> {
        let output = Command::new("apt-get")
            .args(args)
            .arg("-o")
            .arg(format!("Dir={}", self.root.display()))
            .arg("-y")
            .output()
            .context("Failed to execute apt-get")?;
        
        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr);
            bail!("apt-get failed: {}", stderr);
        }
        
        Ok(String::from_utf8_lossy(&output.stdout).to_string())
    }
    
    fn parse_status_file(&self) -> Result<HashMap<String, Package>> {
        let mut packages = HashMap::new();
        
        if !self.status_file.exists() {
            return Ok(packages);
        }
        
        let content = std::fs::read_to_string(&self.status_file)?;
        let mut current = Package::default();
        let mut in_description = false;
        
        for line in content.lines() {
            if line.is_empty() {
                if !current.name.is_empty() {
                    packages.insert(current.name.clone(), current);
                    current = Package::default();
                }
                in_description = false;
                continue;
            }
            
            if in_description && line.starts_with(' ') {
                current.description.push('\n');
                current.description.push_str(line.trim());
                continue;
            }
            
            in_description = false;
            
            if let Some((key, value)) = line.split_once(':') {
                let key = key.trim();
                let value = value.trim();
                
                match key {
                    "Package" => current.name = value.to_string(),
                    "Version" => current.version = value.to_string(),
                    "Description" => {
                        current.description = value.to_string();
                        in_description = true;
                    }
                    "Installed-Size" => {
                        current.installed_size = value.parse::<u64>().unwrap_or(0) * 1024;
                    }
                    "Depends" => {
                        current.dependencies = parse_depends(value);
                    }
                    "Provides" => {
                        current.provides = value.split(',')
                            .map(|s| s.trim().to_string())
                            .collect();
                    }
                    "Conflicts" => {
                        current.conflicts = value.split(',')
                            .map(|s| s.trim().to_string())
                            .collect();
                    }
                    "Status" => {
                        // Check if package is installed
                        if !value.contains("installed") {
                            current.name.clear(); // Mark as not installed
                        }
                    }
                    _ => {}
                }
            }
        }
        
        // Don't forget the last package
        if !current.name.is_empty() {
            packages.insert(current.name.clone(), current);
        }
        
        Ok(packages)
    }
    
    fn parse_package_info(&self, output: &str) -> Option<Package> {
        let mut pkg = Package::default();
        
        for line in output.lines() {
            if let Some((key, value)) = line.split_once(':') {
                let key = key.trim();
                let value = value.trim();
                
                match key {
                    "Package" => pkg.name = value.to_string(),
                    "Version" => pkg.version = value.to_string(),
                    "Description" => pkg.description = value.to_string(),
                    "Size" => pkg.size = value.parse().unwrap_or(0),
                    "Installed-Size" => {
                        pkg.installed_size = value.parse::<u64>().unwrap_or(0) * 1024;
                    }
                    "Depends" => pkg.dependencies = parse_depends(value),
                    _ => {}
                }
            }
        }
        
        if pkg.name.is_empty() {
            None
        } else {
            Some(pkg)
        }
    }
}

impl Backend for DpkgBackend {
    fn name(&self) -> &str {
        "dpkg"
    }
    
    fn init(&mut self) -> Result<()> {
        std::fs::create_dir_all(&self.db_path)?;
        std::fs::create_dir_all(&self.cache_path)?;
        std::fs::create_dir_all(self.db_path.join("info"))?;
        std::fs::create_dir_all(self.db_path.join("updates"))?;
        
        // Create empty status file if not exists
        if !self.status_file.exists() {
            std::fs::write(&self.status_file, "")?;
        }
        
        Ok(())
    }
    
    fn install(&self, package: &Package) -> Result<()> {
        // Try apt-get first
        if let Ok(_) = self.run_apt(&["install", &package.name]) {
            return Ok(());
        }
        
        // Fall back to dpkg if we have a .deb file
        let deb_path = self.cache_path.join(format!("{}_{}.deb", package.name, package.version));
        if deb_path.exists() {
            self.run_dpkg(&["-i", deb_path.to_str().unwrap()])?;
        } else {
            bail!("Package {} not found in cache", package.name);
        }
        
        Ok(())
    }
    
    fn remove(&self, name: &str) -> Result<()> {
        self.run_dpkg(&["-r", name])?;
        Ok(())
    }
    
    fn is_installed(&self, name: &str) -> Result<bool> {
        let packages = self.parse_status_file()?;
        Ok(packages.contains_key(name))
    }
    
    fn get_installed(&self, name: &str) -> Result<Option<Package>> {
        let packages = self.parse_status_file()?;
        Ok(packages.get(name).cloned())
    }
    
    fn list_installed(&self) -> Result<Vec<Package>> {
        let packages = self.parse_status_file()?;
        Ok(packages.into_values().collect())
    }
    
    fn search(&self, query: &str) -> Result<Vec<Package>> {
        let output = Command::new("apt-cache")
            .args(["search", query])
            .output()
            .context("Failed to execute apt-cache")?;
        
        let mut packages = Vec::new();
        let stdout = String::from_utf8_lossy(&output.stdout);
        
        for line in stdout.lines() {
            if let Some((name, desc)) = line.split_once(" - ") {
                packages.push(Package {
                    name: name.to_string(),
                    description: desc.to_string(),
                    ..Default::default()
                });
            }
        }
        
        Ok(packages)
    }
    
    fn get_package(&self, name: &str) -> Result<Option<Package>> {
        let output = Command::new("apt-cache")
            .args(["show", name])
            .output()
            .context("Failed to execute apt-cache")?;
        
        let stdout = String::from_utf8_lossy(&output.stdout);
        Ok(self.parse_package_info(&stdout))
    }
    
    fn sync(&self) -> Result<()> {
        self.run_apt(&["update"])?;
        Ok(())
    }
    
    fn upgrade(&self) -> Result<Vec<Package>> {
        self.run_apt(&["upgrade"])?;
        Ok(Vec::new())
    }
    
    fn get_dependencies(&self, name: &str) -> Result<Vec<String>> {
        let output = Command::new("apt-cache")
            .args(["depends", name])
            .output()
            .context("Failed to execute apt-cache")?;
        
        let mut deps = Vec::new();
        let stdout = String::from_utf8_lossy(&output.stdout);
        
        for line in stdout.lines() {
            let line = line.trim();
            if line.starts_with("Depends:") {
                if let Some(dep) = line.strip_prefix("Depends:") {
                    deps.push(dep.trim().to_string());
                }
            }
        }
        
        Ok(deps)
    }
    
    fn verify(&self, package: &Package) -> Result<bool> {
        let output = self.run_dpkg(&["-V", &package.name]);
        Ok(output.is_ok())
    }
}

fn parse_depends(value: &str) -> Vec<String> {
    value.split(',')
        .map(|s| {
            // Handle alternatives (a | b) - take first
            let s = s.split('|').next().unwrap_or(s);
            // Remove version constraints
            s.split(|c| c == '(' || c == ')' || c == ' ')
                .next()
                .unwrap_or(s)
                .trim()
                .to_string()
        })
        .filter(|s| !s.is_empty())
        .collect()
}
