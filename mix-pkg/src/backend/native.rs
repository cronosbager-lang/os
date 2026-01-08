// MIXOS Native Package Manager Backend
use anyhow::{Result, Context, bail};
use std::path::{Path, PathBuf};
use std::collections::HashMap;
use std::fs;

use super::Backend;
use crate::core::package::Package;
use crate::utils::{download, compression, crypto};

/// Native MIXOS package backend
pub struct NativeBackend {
    root: PathBuf,
    db_path: PathBuf,
    cache_path: PathBuf,
    installed_path: PathBuf,
    repos: Vec<Repository>,
}

#[derive(Debug, Clone)]
struct Repository {
    name: String,
    url: String,
    packages: HashMap<String, Package>,
}

impl NativeBackend {
    pub fn new(root: &Path) -> Result<Self> {
        let root = root.to_path_buf();
        let db_path = root.join("var/lib/mix-pkg/db");
        let cache_path = root.join("var/cache/mix-pkg");
        let installed_path = root.join("var/lib/mix-pkg/installed");
        
        Ok(Self {
            root,
            db_path,
            cache_path,
            installed_path,
            repos: Vec::new(),
        })
    }
    
    fn load_repos(&mut self) -> Result<()> {
        let repos_file = self.root.join("etc/mixos/repositories.toml");
        if !repos_file.exists() {
            return Ok(());
        }
        
        let content = fs::read_to_string(&repos_file)?;
        let config: toml::Value = toml::from_str(&content)?;
        
        if let Some(repos) = config.get("repositories").and_then(|r| r.as_array()) {
            for repo in repos {
                let name = repo.get("name")
                    .and_then(|n| n.as_str())
                    .unwrap_or("unknown")
                    .to_string();
                let url = repo.get("url")
                    .and_then(|u| u.as_str())
                    .unwrap_or("")
                    .to_string();
                
                if !url.is_empty() {
                    self.repos.push(Repository {
                        name,
                        url,
                        packages: HashMap::new(),
                    });
                }
            }
        }
        
        Ok(())
    }
    
    fn load_repo_db(&mut self, repo_idx: usize) -> Result<()> {
        let repo = &self.repos[repo_idx];
        let db_file = self.db_path.join(format!("{}.db", repo.name));
        
        if !db_file.exists() {
            return Ok(());
        }
        
        let content = fs::read_to_string(&db_file)?;
        let packages: Vec<Package> = serde_json::from_str(&content)
            .unwrap_or_default();
        
        let repo = &mut self.repos[repo_idx];
        for pkg in packages {
            repo.packages.insert(pkg.name.clone(), pkg);
        }
        
        Ok(())
    }
    
    fn get_installed_packages(&self) -> Result<HashMap<String, Package>> {
        let mut packages = HashMap::new();
        
        if !self.installed_path.exists() {
            return Ok(packages);
        }
        
        for entry in fs::read_dir(&self.installed_path)? {
            let entry = entry?;
            if !entry.file_type()?.is_dir() {
                continue;
            }
            
            let pkg_dir = entry.path();
            let info_file = pkg_dir.join("info.json");
            
            if info_file.exists() {
                let content = fs::read_to_string(&info_file)?;
                if let Ok(pkg) = serde_json::from_str::<Package>(&content) {
                    packages.insert(pkg.name.clone(), pkg);
                }
            } else {
                // Legacy format
                let name = entry.file_name().to_string_lossy().to_string();
                let version = fs::read_to_string(pkg_dir.join("version"))
                    .unwrap_or_default()
                    .trim()
                    .to_string();
                let description = fs::read_to_string(pkg_dir.join("desc"))
                    .unwrap_or_default()
                    .trim()
                    .to_string();
                
                packages.insert(name.clone(), Package {
                    name,
                    version,
                    description,
                    ..Default::default()
                });
            }
        }
        
        Ok(packages)
    }
    
    fn download_package(&self, pkg: &Package) -> Result<PathBuf> {
        let filename = format!("{}-{}.pkg.tar.zst", pkg.name, pkg.version);
        let cache_file = self.cache_path.join(&filename);
        
        if cache_file.exists() {
            // Verify checksum
            if !pkg.checksum.is_empty() {
                let computed = crypto::sha256_file(&cache_file)?;
                if computed == pkg.checksum {
                    return Ok(cache_file);
                }
                // Checksum mismatch, re-download
                fs::remove_file(&cache_file)?;
            } else {
                return Ok(cache_file);
            }
        }
        
        // Find package URL
        let url = format!("{}/pool/{}/{}", 
            self.repos.first().map(|r| r.url.as_str()).unwrap_or(""),
            pkg.repository,
            filename
        );
        
        fs::create_dir_all(&self.cache_path)?;
        download::download_file(&url, &cache_file)?;
        
        // Verify checksum after download
        if !pkg.checksum.is_empty() {
            let computed = crypto::sha256_file(&cache_file)?;
            if computed != pkg.checksum {
                fs::remove_file(&cache_file)?;
                bail!("Checksum verification failed for {}", pkg.name);
            }
        }
        
        Ok(cache_file)
    }
    
    fn extract_package(&self, pkg_path: &Path, pkg: &Package) -> Result<()> {
        // Extract to root
        compression::extract_tar_zst(pkg_path, &self.root)?;
        
        // Record installation
        let pkg_dir = self.installed_path.join(&pkg.name);
        fs::create_dir_all(&pkg_dir)?;
        
        // Save package info
        let info = serde_json::to_string_pretty(pkg)?;
        fs::write(pkg_dir.join("info.json"), info)?;
        
        // Run post-install script if exists
        let post_install = pkg_dir.join("install.sh");
        if post_install.exists() {
            std::process::Command::new("sh")
                .arg(&post_install)
                .arg("post_install")
                .current_dir(&self.root)
                .status()?;
        }
        
        Ok(())
    }
}

impl Backend for NativeBackend {
    fn name(&self) -> &str {
        "native"
    }
    
    fn init(&mut self) -> Result<()> {
        fs::create_dir_all(&self.db_path)?;
        fs::create_dir_all(&self.cache_path)?;
        fs::create_dir_all(&self.installed_path)?;
        
        self.load_repos()?;
        
        for i in 0..self.repos.len() {
            let _ = self.load_repo_db(i);
        }
        
        Ok(())
    }
    
    fn install(&self, package: &Package) -> Result<()> {
        // Download package
        let pkg_path = self.download_package(package)?;
        
        // Extract and install
        self.extract_package(&pkg_path, package)?;
        
        Ok(())
    }
    
    fn remove(&self, name: &str) -> Result<()> {
        let pkg_dir = self.installed_path.join(name);
        
        if !pkg_dir.exists() {
            bail!("Package {} is not installed", name);
        }
        
        // Read file list
        let files_path = pkg_dir.join("files");
        if files_path.exists() {
            let content = fs::read_to_string(&files_path)?;
            for file in content.lines().rev() {
                let file_path = Path::new(file);
                if file_path.exists() {
                    if file_path.is_dir() {
                        let _ = fs::remove_dir(file_path);
                    } else {
                        let _ = fs::remove_file(file_path);
                    }
                }
            }
        }
        
        // Run pre-remove script
        let pre_remove = pkg_dir.join("install.sh");
        if pre_remove.exists() {
            std::process::Command::new("sh")
                .arg(&pre_remove)
                .arg("pre_remove")
                .current_dir(&self.root)
                .status()?;
        }
        
        // Remove package directory
        fs::remove_dir_all(&pkg_dir)?;
        
        Ok(())
    }
    
    fn is_installed(&self, name: &str) -> Result<bool> {
        Ok(self.installed_path.join(name).exists())
    }
    
    fn get_installed(&self, name: &str) -> Result<Option<Package>> {
        let packages = self.get_installed_packages()?;
        Ok(packages.get(name).cloned())
    }
    
    fn list_installed(&self) -> Result<Vec<Package>> {
        let packages = self.get_installed_packages()?;
        Ok(packages.into_values().collect())
    }
    
    fn search(&self, query: &str) -> Result<Vec<Package>> {
        let query = query.to_lowercase();
        let mut results = Vec::new();
        
        for repo in &self.repos {
            for pkg in repo.packages.values() {
                if pkg.name.to_lowercase().contains(&query) ||
                   pkg.description.to_lowercase().contains(&query) {
                    results.push(pkg.clone());
                }
            }
        }
        
        Ok(results)
    }
    
    fn get_package(&self, name: &str) -> Result<Option<Package>> {
        for repo in &self.repos {
            if let Some(pkg) = repo.packages.get(name) {
                return Ok(Some(pkg.clone()));
            }
        }
        Ok(None)
    }
    
    fn sync(&self) -> Result<()> {
        fs::create_dir_all(&self.db_path)?;
        
        for repo in &self.repos {
            let db_url = format!("{}/db/{}.db", repo.url, repo.name);
            let db_file = self.db_path.join(format!("{}.db", repo.name));
            
            if let Err(e) = download::download_file(&db_url, &db_file) {
                eprintln!("Warning: Failed to sync {}: {}", repo.name, e);
            }
        }
        
        Ok(())
    }
    
    fn upgrade(&self) -> Result<Vec<Package>> {
        let installed = self.get_installed_packages()?;
        let mut upgraded = Vec::new();
        
        for (name, installed_pkg) in &installed {
            if let Ok(Some(repo_pkg)) = self.get_package(name) {
                if version_compare(&repo_pkg.version, &installed_pkg.version) > 0 {
                    self.install(&repo_pkg)?;
                    upgraded.push(repo_pkg);
                }
            }
        }
        
        Ok(upgraded)
    }
    
    fn get_dependencies(&self, name: &str) -> Result<Vec<String>> {
        if let Some(pkg) = self.get_package(name)? {
            Ok(pkg.dependencies)
        } else {
            Ok(Vec::new())
        }
    }
    
    fn verify(&self, package: &Package) -> Result<bool> {
        let pkg_dir = self.installed_path.join(&package.name);
        let files_path = pkg_dir.join("files");
        
        if !files_path.exists() {
            return Ok(false);
        }
        
        let content = fs::read_to_string(&files_path)?;
        for file in content.lines() {
            if !Path::new(file).exists() {
                return Ok(false);
            }
        }
        
        Ok(true)
    }
}

/// Simple version comparison
fn version_compare(a: &str, b: &str) -> i32 {
    let a_parts: Vec<&str> = a.split(|c| c == '.' || c == '-').collect();
    let b_parts: Vec<&str> = b.split(|c| c == '.' || c == '-').collect();
    
    for i in 0..a_parts.len().max(b_parts.len()) {
        let a_part = a_parts.get(i).unwrap_or(&"0");
        let b_part = b_parts.get(i).unwrap_or(&"0");
        
        // Try numeric comparison first
        if let (Ok(a_num), Ok(b_num)) = (a_part.parse::<u64>(), b_part.parse::<u64>()) {
            if a_num > b_num {
                return 1;
            } else if a_num < b_num {
                return -1;
            }
        } else {
            // Fall back to string comparison
            match a_part.cmp(b_part) {
                std::cmp::Ordering::Greater => return 1,
                std::cmp::Ordering::Less => return -1,
                std::cmp::Ordering::Equal => {}
            }
        }
    }
    
    0
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_version_compare() {
        assert_eq!(version_compare("1.0.0", "1.0.0"), 0);
        assert_eq!(version_compare("1.0.1", "1.0.0"), 1);
        assert_eq!(version_compare("1.0.0", "1.0.1"), -1);
        assert_eq!(version_compare("2.0.0", "1.9.9"), 1);
        assert_eq!(version_compare("1.10.0", "1.9.0"), 1);
    }
}
