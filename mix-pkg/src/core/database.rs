use anyhow::{Result, Context};
use std::collections::HashMap;
use std::path::PathBuf;

use crate::config::Config;
use super::package::Package;

pub struct Database {
    packages: HashMap<String, Package>,
    installed: HashMap<String, String>, // name -> version
    db_path: PathBuf,
}

impl Database {
    pub fn load(config: &Config) -> Result<Self> {
        let db_path = config.db_path();
        std::fs::create_dir_all(&db_path)?;
        
        let mut packages = HashMap::new();
        let mut installed = HashMap::new();
        
        // Load package database for each repository
        for repo in &config.repositories {
            if !repo.enabled {
                continue;
            }
            
            let repo_db = db_path.join(format!("{}.db", repo.name));
            if repo_db.exists() {
                if let Ok(content) = std::fs::read_to_string(&repo_db) {
                    for pkg_info in content.split("\n\n") {
                        if pkg_info.trim().is_empty() {
                            continue;
                        }
                        if let Ok(pkg) = Package::from_pkginfo(pkg_info, &repo.name) {
                            packages.insert(pkg.name.clone(), pkg);
                        }
                    }
                }
            }
        }
        
        // Load installed packages
        let installed_dir = PathBuf::from("/var/lib/mix-pkg/installed");
        if installed_dir.exists() {
            for entry in std::fs::read_dir(&installed_dir)? {
                let entry = entry?;
                if entry.file_type()?.is_dir() {
                    let name = entry.file_name().to_string_lossy().to_string();
                    let version_file = entry.path().join("version");
                    if let Ok(version) = std::fs::read_to_string(&version_file) {
                        installed.insert(name, version.trim().to_string());
                    }
                }
            }
        }
        
        Ok(Database {
            packages,
            installed,
            db_path,
        })
    }
    
    pub fn find_package(&self, name: &str) -> Option<Package> {
        self.packages.get(name).cloned()
    }
    
    pub fn is_installed(&self, name: &str) -> bool {
        self.installed.contains_key(name)
    }
    
    pub fn search(&self, query: &str) -> Vec<Package> {
        let query = query.to_lowercase();
        self.packages
            .values()
            .filter(|pkg| {
                pkg.name.to_lowercase().contains(&query) ||
                pkg.description.to_lowercase().contains(&query)
            })
            .cloned()
            .collect()
    }
    
    pub fn list_installed(&self) -> Vec<(String, String)> {
        let mut list: Vec<_> = self.installed
            .iter()
            .map(|(k, v)| (k.clone(), v.clone()))
            .collect();
        list.sort_by(|a, b| a.0.cmp(&b.0));
        list
    }
    
    pub fn resolve_dependencies(&self, pkg: &Package) -> Result<Vec<Package>> {
        let mut deps = Vec::new();
        let mut visited = std::collections::HashSet::new();
        
        self.resolve_deps_recursive(pkg, &mut deps, &mut visited)?;
        
        Ok(deps)
    }
    
    fn resolve_deps_recursive(
        &self,
        pkg: &Package,
        deps: &mut Vec<Package>,
        visited: &mut std::collections::HashSet<String>,
    ) -> Result<()> {
        for dep_name in &pkg.dependencies {
            // Handle versioned dependencies (e.g., "glibc>=2.17")
            let dep_name = dep_name.split(|c| c == '>' || c == '<' || c == '=')
                .next()
                .unwrap_or(dep_name);
            
            if visited.contains(dep_name) || self.is_installed(dep_name) {
                continue;
            }
            
            visited.insert(dep_name.to_string());
            
            if let Some(dep_pkg) = self.find_package(dep_name) {
                self.resolve_deps_recursive(&dep_pkg, deps, visited)?;
                deps.push(dep_pkg);
            }
        }
        
        Ok(())
    }
    
    pub fn get_dependents(&self, pkg_name: &str) -> Vec<String> {
        self.installed
            .keys()
            .filter(|name| {
                if let Some(pkg) = self.packages.get(*name) {
                    pkg.dependencies.iter().any(|d| {
                        d.split(|c| c == '>' || c == '<' || c == '=')
                            .next()
                            .unwrap_or(d) == pkg_name
                    })
                } else {
                    false
                }
            })
            .cloned()
            .collect()
    }
    
    pub fn get_upgradable(&self) -> Vec<(String, String, String)> {
        let mut upgrades = Vec::new();
        
        for (name, installed_ver) in &self.installed {
            if let Some(pkg) = self.packages.get(name) {
                if pkg.version != *installed_ver {
                    // Simple version comparison (could be improved)
                    if pkg.version > *installed_ver {
                        upgrades.push((name.clone(), installed_ver.clone(), pkg.version.clone()));
                    }
                }
            }
        }
        
        upgrades
    }
}

pub fn record_installation(pkg: &Package) -> Result<()> {
    let pkg_dir = format!("/var/lib/mix-pkg/installed/{}", pkg.name);
    std::fs::create_dir_all(&pkg_dir)?;
    
    std::fs::write(format!("{}/version", pkg_dir), &pkg.version)?;
    std::fs::write(format!("{}/desc", pkg_dir), &pkg.description)?;
    
    if !pkg.dependencies.is_empty() {
        std::fs::write(format!("{}/depends", pkg_dir), pkg.dependencies.join("\n"))?;
    }
    
    Ok(())
}
