// ALPM (Arch Linux Package Manager) Backend
use anyhow::{Result, Context, bail};
use std::path::{Path, PathBuf};
use std::process::Command;

use super::Backend;
use crate::core::package::Package;

/// ALPM Backend implementation
pub struct AlpmBackend {
    root: PathBuf,
    db_path: PathBuf,
    cache_path: PathBuf,
}

impl AlpmBackend {
    pub fn new(root: &Path) -> Result<Self> {
        let root = root.to_path_buf();
        let db_path = root.join("var/lib/pacman");
        let cache_path = root.join("var/cache/pacman/pkg");
        
        Ok(Self {
            root,
            db_path,
            cache_path,
        })
    }
    
    fn run_pacman(&self, args: &[&str]) -> Result<String> {
        let output = Command::new("pacman")
            .args(args)
            .arg("--root")
            .arg(&self.root)
            .arg("--dbpath")
            .arg(&self.db_path)
            .arg("--cachedir")
            .arg(&self.cache_path)
            .arg("--noconfirm")
            .output()
            .context("Failed to execute pacman")?;
        
        if !output.status.success() {
            let stderr = String::from_utf8_lossy(&output.stderr);
            bail!("pacman failed: {}", stderr);
        }
        
        Ok(String::from_utf8_lossy(&output.stdout).to_string())
    }
    
    fn parse_package_info(&self, output: &str) -> Option<Package> {
        let mut name = String::new();
        let mut version = String::new();
        let mut description = String::new();
        let mut size: u64 = 0;
        let mut depends = Vec::new();
        
        for line in output.lines() {
            if let Some((key, value)) = line.split_once(':') {
                let key = key.trim();
                let value = value.trim();
                
                match key {
                    "Name" => name = value.to_string(),
                    "Version" => version = value.to_string(),
                    "Description" => description = value.to_string(),
                    "Installed Size" => {
                        size = parse_size(value).unwrap_or(0);
                    }
                    "Depends On" => {
                        if value != "None" {
                            depends = value.split_whitespace()
                                .map(|s| s.to_string())
                                .collect();
                        }
                    }
                    _ => {}
                }
            }
        }
        
        if name.is_empty() {
            return None;
        }
        
        Some(Package {
            name,
            version,
            description,
            size,
            depends,
            ..Default::default()
        })
    }
}

impl Backend for AlpmBackend {
    fn name(&self) -> &str {
        "alpm"
    }
    
    fn init(&mut self) -> Result<()> {
        // Create directories
        std::fs::create_dir_all(&self.db_path)?;
        std::fs::create_dir_all(&self.cache_path)?;
        
        // Initialize pacman keyring if needed
        if !self.db_path.join("sync").exists() {
            self.sync()?;
        }
        
        Ok(())
    }
    
    fn install(&self, package: &Package) -> Result<()> {
        self.run_pacman(&["-S", &package.name])?;
        Ok(())
    }
    
    fn remove(&self, name: &str) -> Result<()> {
        self.run_pacman(&["-R", name])?;
        Ok(())
    }
    
    fn is_installed(&self, name: &str) -> Result<bool> {
        let result = self.run_pacman(&["-Q", name]);
        Ok(result.is_ok())
    }
    
    fn get_installed(&self, name: &str) -> Result<Option<Package>> {
        let output = self.run_pacman(&["-Qi", name])?;
        Ok(self.parse_package_info(&output))
    }
    
    fn list_installed(&self) -> Result<Vec<Package>> {
        let output = self.run_pacman(&["-Q"])?;
        let mut packages = Vec::new();
        
        for line in output.lines() {
            let parts: Vec<&str> = line.split_whitespace().collect();
            if parts.len() >= 2 {
                packages.push(Package {
                    name: parts[0].to_string(),
                    version: parts[1].to_string(),
                    ..Default::default()
                });
            }
        }
        
        Ok(packages)
    }
    
    fn search(&self, query: &str) -> Result<Vec<Package>> {
        let output = self.run_pacman(&["-Ss", query])?;
        let mut packages = Vec::new();
        let mut current_name = String::new();
        let mut current_version = String::new();
        
        for line in output.lines() {
            if line.starts_with(' ') {
                // Description line
                if !current_name.is_empty() {
                    packages.push(Package {
                        name: current_name.clone(),
                        version: current_version.clone(),
                        description: line.trim().to_string(),
                        ..Default::default()
                    });
                    current_name.clear();
                }
            } else if let Some((repo_name, version)) = line.split_once(' ') {
                // Package line: repo/name version
                if let Some((_, name)) = repo_name.split_once('/') {
                    current_name = name.to_string();
                    current_version = version.to_string();
                }
            }
        }
        
        Ok(packages)
    }
    
    fn get_package(&self, name: &str) -> Result<Option<Package>> {
        let output = self.run_pacman(&["-Si", name])?;
        Ok(self.parse_package_info(&output))
    }
    
    fn sync(&self) -> Result<()> {
        self.run_pacman(&["-Sy"])?;
        Ok(())
    }
    
    fn upgrade(&self) -> Result<Vec<Package>> {
        let output = self.run_pacman(&["-Syu"])?;
        // Parse upgraded packages from output
        let packages = Vec::new();
        // TODO: Parse output for upgraded packages
        Ok(packages)
    }
    
    fn get_dependencies(&self, name: &str) -> Result<Vec<String>> {
        let output = self.run_pacman(&["-Si", name])?;
        
        for line in output.lines() {
            if let Some((key, value)) = line.split_once(':') {
                if key.trim() == "Depends On" && value.trim() != "None" {
                    return Ok(value.split_whitespace()
                        .map(|s| s.to_string())
                        .collect());
                }
            }
        }
        
        Ok(Vec::new())
    }
    
    fn verify(&self, package: &Package) -> Result<bool> {
        let result = self.run_pacman(&["-Qk", &package.name]);
        Ok(result.is_ok())
    }
}

fn parse_size(s: &str) -> Option<u64> {
    let s = s.trim();
    let (num, unit) = if s.ends_with("KiB") {
        (s.trim_end_matches("KiB").trim(), 1024u64)
    } else if s.ends_with("MiB") {
        (s.trim_end_matches("MiB").trim(), 1024 * 1024)
    } else if s.ends_with("GiB") {
        (s.trim_end_matches("GiB").trim(), 1024 * 1024 * 1024)
    } else if s.ends_with("B") {
        (s.trim_end_matches("B").trim(), 1)
    } else {
        (s, 1)
    };
    
    num.parse::<f64>().ok().map(|n| (n * unit as f64) as u64)
}
