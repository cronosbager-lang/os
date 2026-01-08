// Dependency Resolution Module
use anyhow::{Result, bail};
use std::collections::{HashMap, HashSet, VecDeque};

use super::package::Package;

/// Dependency resolver with cycle detection and conflict handling
pub struct DependencyResolver {
    packages: HashMap<String, Package>,
    installed: HashSet<String>,
    provides: HashMap<String, String>, // virtual -> real package
}

#[derive(Debug, Clone)]
pub struct ResolutionResult {
    pub to_install: Vec<Package>,
    pub to_remove: Vec<String>,
    pub conflicts: Vec<Conflict>,
    pub missing: Vec<String>,
}

#[derive(Debug, Clone)]
pub struct Conflict {
    pub package_a: String,
    pub package_b: String,
    pub reason: String,
}

impl DependencyResolver {
    pub fn new() -> Self {
        Self {
            packages: HashMap::new(),
            installed: HashSet::new(),
            provides: HashMap::new(),
        }
    }
    
    pub fn add_package(&mut self, pkg: Package) {
        // Register provides
        for provided in &pkg.provides {
            self.provides.insert(provided.clone(), pkg.name.clone());
        }
        self.packages.insert(pkg.name.clone(), pkg);
    }
    
    pub fn mark_installed(&mut self, name: &str) {
        self.installed.insert(name.to_string());
    }
    
    pub fn resolve(&self, targets: &[String]) -> Result<ResolutionResult> {
        let mut result = ResolutionResult {
            to_install: Vec::new(),
            to_remove: Vec::new(),
            conflicts: Vec::new(),
            missing: Vec::new(),
        };
        
        let mut visited = HashSet::new();
        let mut in_stack = HashSet::new();
        let mut order = Vec::new();
        
        for target in targets {
            self.resolve_recursive(target, &mut visited, &mut in_stack, &mut order, &mut result)?;
        }
        
        // Filter out already installed packages
        result.to_install = order.into_iter()
            .filter(|pkg| !self.installed.contains(&pkg.name))
            .collect();
        
        // Check for conflicts
        self.check_conflicts(&result.to_install, &mut result.conflicts);
        
        Ok(result)
    }
    
    fn resolve_recursive(
        &self,
        name: &str,
        visited: &mut HashSet<String>,
        in_stack: &mut HashSet<String>,
        order: &mut Vec<Package>,
        result: &mut ResolutionResult,
    ) -> Result<()> {
        // Check for cycles
        if in_stack.contains(name) {
            bail!("Circular dependency detected involving {}", name);
        }
        
        if visited.contains(name) {
            return Ok(());
        }
        
        // Resolve virtual packages
        let real_name = self.provides.get(name).unwrap_or(&name.to_string()).clone();
        
        let pkg = match self.packages.get(&real_name) {
            Some(p) => p.clone(),
            None => {
                if !self.installed.contains(&real_name) {
                    result.missing.push(name.to_string());
                }
                return Ok(());
            }
        };
        
        visited.insert(name.to_string());
        in_stack.insert(name.to_string());
        
        // Resolve dependencies first
        for dep in &pkg.dependencies {
            let dep_name = parse_dependency_name(dep);
            self.resolve_recursive(&dep_name, visited, in_stack, order, result)?;
        }
        
        in_stack.remove(name);
        order.push(pkg);
        
        Ok(())
    }
    
    fn check_conflicts(&self, packages: &[Package], conflicts: &mut Vec<Conflict>) {
        let mut to_install: HashSet<String> = packages.iter()
            .map(|p| p.name.clone())
            .collect();
        
        // Add installed packages
        to_install.extend(self.installed.iter().cloned());
        
        for pkg in packages {
            for conflict in &pkg.conflicts {
                let conflict_name = parse_dependency_name(conflict);
                if to_install.contains(&conflict_name) {
                    conflicts.push(Conflict {
                        package_a: pkg.name.clone(),
                        package_b: conflict_name,
                        reason: format!("{} conflicts with {}", pkg.name, conflict),
                    });
                }
            }
        }
    }
    
    /// Resolve removal - find packages that depend on the target
    pub fn resolve_removal(&self, targets: &[String]) -> Result<Vec<String>> {
        let mut to_remove = HashSet::new();
        let mut queue: VecDeque<String> = targets.iter().cloned().collect();
        
        while let Some(name) = queue.pop_front() {
            if to_remove.contains(&name) {
                continue;
            }
            to_remove.insert(name.clone());
            
            // Find packages that depend on this one
            for (pkg_name, pkg) in &self.packages {
                if !self.installed.contains(pkg_name) {
                    continue;
                }
                
                for dep in &pkg.dependencies {
                    let dep_name = parse_dependency_name(dep);
                    if dep_name == name && !to_remove.contains(pkg_name) {
                        queue.push_back(pkg_name.clone());
                    }
                }
            }
        }
        
        Ok(to_remove.into_iter().collect())
    }
    
    /// Check if a package can be safely removed
    pub fn can_remove(&self, name: &str) -> Result<(bool, Vec<String>)> {
        let mut dependents = Vec::new();
        
        for (pkg_name, pkg) in &self.packages {
            if !self.installed.contains(pkg_name) || pkg_name == name {
                continue;
            }
            
            for dep in &pkg.dependencies {
                let dep_name = parse_dependency_name(dep);
                if dep_name == name {
                    dependents.push(pkg_name.clone());
                    break;
                }
            }
        }
        
        Ok((dependents.is_empty(), dependents))
    }
    
    /// Get upgrade candidates
    pub fn get_upgrades(&self, available: &HashMap<String, Package>) -> Vec<(Package, Package)> {
        let mut upgrades = Vec::new();
        
        for name in &self.installed {
            if let (Some(installed), Some(available)) = (
                self.packages.get(name),
                available.get(name)
            ) {
                if version_newer(&available.version, &installed.version) {
                    upgrades.push((installed.clone(), available.clone()));
                }
            }
        }
        
        upgrades
    }
}

impl Default for DependencyResolver {
    fn default() -> Self {
        Self::new()
    }
}

/// Parse dependency name from version constraint
fn parse_dependency_name(dep: &str) -> String {
    dep.split(|c| c == '>' || c == '<' || c == '=' || c == '(' || c == ' ')
        .next()
        .unwrap_or(dep)
        .trim()
        .to_string()
}

/// Check if version a is newer than version b
fn version_newer(a: &str, b: &str) -> bool {
    let a_parts: Vec<u64> = a.split(|c| c == '.' || c == '-')
        .filter_map(|s| s.parse().ok())
        .collect();
    let b_parts: Vec<u64> = b.split(|c| c == '.' || c == '-')
        .filter_map(|s| s.parse().ok())
        .collect();
    
    for i in 0..a_parts.len().max(b_parts.len()) {
        let a_part = a_parts.get(i).unwrap_or(&0);
        let b_part = b_parts.get(i).unwrap_or(&0);
        
        if a_part > b_part {
            return true;
        } else if a_part < b_part {
            return false;
        }
    }
    
    false
}

/// Version constraint checking
pub struct VersionConstraint {
    pub operator: ConstraintOp,
    pub version: String,
}

#[derive(Debug, Clone, PartialEq)]
pub enum ConstraintOp {
    Any,
    Eq,
    Lt,
    Le,
    Gt,
    Ge,
}

impl VersionConstraint {
    pub fn parse(s: &str) -> Option<(String, Self)> {
        let s = s.trim();
        
        // Check for version constraint
        if let Some(idx) = s.find(|c| c == '>' || c == '<' || c == '=') {
            let name = s[..idx].trim().to_string();
            let rest = &s[idx..];
            
            let (op, version) = if rest.starts_with(">=") {
                (ConstraintOp::Ge, rest[2..].trim())
            } else if rest.starts_with("<=") {
                (ConstraintOp::Le, rest[2..].trim())
            } else if rest.starts_with("==") || rest.starts_with("=") {
                let v = rest.trim_start_matches('=').trim();
                (ConstraintOp::Eq, v)
            } else if rest.starts_with('>') {
                (ConstraintOp::Gt, rest[1..].trim())
            } else if rest.starts_with('<') {
                (ConstraintOp::Lt, rest[1..].trim())
            } else {
                return None;
            };
            
            Some((name, Self {
                operator: op,
                version: version.to_string(),
            }))
        } else {
            Some((s.to_string(), Self {
                operator: ConstraintOp::Any,
                version: String::new(),
            }))
        }
    }
    
    pub fn satisfied_by(&self, version: &str) -> bool {
        match self.operator {
            ConstraintOp::Any => true,
            ConstraintOp::Eq => version == self.version,
            ConstraintOp::Lt => version_newer(&self.version, version),
            ConstraintOp::Le => version == self.version || version_newer(&self.version, version),
            ConstraintOp::Gt => version_newer(version, &self.version),
            ConstraintOp::Ge => version == self.version || version_newer(version, &self.version),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_parse_dependency_name() {
        assert_eq!(parse_dependency_name("glibc>=2.17"), "glibc");
        assert_eq!(parse_dependency_name("openssl"), "openssl");
        assert_eq!(parse_dependency_name("python (>= 3.8)"), "python");
    }
    
    #[test]
    fn test_version_newer() {
        assert!(version_newer("2.0.0", "1.0.0"));
        assert!(version_newer("1.1.0", "1.0.0"));
        assert!(!version_newer("1.0.0", "1.0.0"));
        assert!(!version_newer("1.0.0", "2.0.0"));
    }
    
    #[test]
    fn test_version_constraint() {
        let (name, constraint) = VersionConstraint::parse("glibc>=2.17").unwrap();
        assert_eq!(name, "glibc");
        assert_eq!(constraint.operator, ConstraintOp::Ge);
        assert!(constraint.satisfied_by("2.17"));
        assert!(constraint.satisfied_by("2.18"));
        assert!(!constraint.satisfied_by("2.16"));
    }
}
