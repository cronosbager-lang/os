// Transaction Module - Atomic package operations
use anyhow::{Result, Context, bail};
use std::path::{Path, PathBuf};
use std::fs;
use std::collections::HashSet;

use super::package::Package;

/// Transaction state
#[derive(Debug, Clone, PartialEq)]
pub enum TransactionState {
    Pending,
    Downloading,
    Installing,
    Configuring,
    Completed,
    Failed,
    RolledBack,
}

/// Transaction operation type
#[derive(Debug, Clone)]
pub enum TransactionOp {
    Install(Package),
    Remove(String),
    Upgrade(Package, Package), // old, new
}

/// Package transaction for atomic operations
pub struct Transaction {
    id: String,
    state: TransactionState,
    operations: Vec<TransactionOp>,
    completed_ops: Vec<TransactionOp>,
    backup_dir: PathBuf,
    root: PathBuf,
}

impl Transaction {
    pub fn new(root: &Path) -> Self {
        let id = format!("txn_{}", chrono_timestamp());
        let backup_dir = root.join("var/lib/mix-pkg/transactions").join(&id);
        
        Self {
            id,
            state: TransactionState::Pending,
            operations: Vec::new(),
            completed_ops: Vec::new(),
            backup_dir,
            root: root.to_path_buf(),
        }
    }
    
    pub fn id(&self) -> &str {
        &self.id
    }
    
    pub fn state(&self) -> &TransactionState {
        &self.state
    }
    
    pub fn add_install(&mut self, pkg: Package) {
        self.operations.push(TransactionOp::Install(pkg));
    }
    
    pub fn add_remove(&mut self, name: String) {
        self.operations.push(TransactionOp::Remove(name));
    }
    
    pub fn add_upgrade(&mut self, old: Package, new: Package) {
        self.operations.push(TransactionOp::Upgrade(old, new));
    }
    
    pub fn operations(&self) -> &[TransactionOp] {
        &self.operations
    }
    
    /// Prepare transaction - create backup directory and validate
    pub fn prepare(&mut self) -> Result<()> {
        fs::create_dir_all(&self.backup_dir)?;
        
        // Validate all operations
        for op in &self.operations {
            match op {
                TransactionOp::Install(pkg) => {
                    if pkg.name.is_empty() {
                        bail!("Invalid package: empty name");
                    }
                }
                TransactionOp::Remove(name) => {
                    let installed_dir = self.root.join("var/lib/mix-pkg/installed").join(name);
                    if !installed_dir.exists() {
                        bail!("Package {} is not installed", name);
                    }
                }
                TransactionOp::Upgrade(old, new) => {
                    if old.name != new.name {
                        bail!("Upgrade package name mismatch: {} vs {}", old.name, new.name);
                    }
                }
            }
        }
        
        Ok(())
    }
    
    /// Execute the transaction
    pub fn execute<F>(&mut self, mut callback: F) -> Result<()>
    where
        F: FnMut(&TransactionOp, &TransactionState),
    {
        self.state = TransactionState::Downloading;
        
        for op in self.operations.clone() {
            callback(&op, &self.state);
            
            match &op {
                TransactionOp::Install(pkg) => {
                    self.state = TransactionState::Installing;
                    callback(&op, &self.state);
                    
                    // Backup would happen here if package exists
                    self.backup_if_exists(&pkg.name)?;
                }
                TransactionOp::Remove(name) => {
                    self.state = TransactionState::Installing;
                    callback(&op, &self.state);
                    
                    // Backup before removal
                    self.backup_package(name)?;
                }
                TransactionOp::Upgrade(old, _new) => {
                    self.state = TransactionState::Installing;
                    callback(&op, &self.state);
                    
                    // Backup old version
                    self.backup_package(&old.name)?;
                }
            }
            
            self.completed_ops.push(op);
        }
        
        self.state = TransactionState::Configuring;
        
        // Run post-transaction hooks
        self.run_hooks()?;
        
        self.state = TransactionState::Completed;
        
        // Cleanup backup on success
        let _ = fs::remove_dir_all(&self.backup_dir);
        
        Ok(())
    }
    
    /// Rollback the transaction
    pub fn rollback(&mut self) -> Result<()> {
        self.state = TransactionState::RolledBack;
        
        // Restore backups in reverse order
        for op in self.completed_ops.iter().rev() {
            match op {
                TransactionOp::Install(pkg) => {
                    // Remove installed package
                    let installed_dir = self.root.join("var/lib/mix-pkg/installed").join(&pkg.name);
                    let _ = fs::remove_dir_all(&installed_dir);
                    
                    // Restore backup if exists
                    self.restore_backup(&pkg.name)?;
                }
                TransactionOp::Remove(name) => {
                    // Restore removed package
                    self.restore_backup(name)?;
                }
                TransactionOp::Upgrade(old, _new) => {
                    // Restore old version
                    self.restore_backup(&old.name)?;
                }
            }
        }
        
        Ok(())
    }
    
    fn backup_if_exists(&self, name: &str) -> Result<()> {
        let installed_dir = self.root.join("var/lib/mix-pkg/installed").join(name);
        if installed_dir.exists() {
            self.backup_package(name)?;
        }
        Ok(())
    }
    
    fn backup_package(&self, name: &str) -> Result<()> {
        let installed_dir = self.root.join("var/lib/mix-pkg/installed").join(name);
        let backup_path = self.backup_dir.join(name);
        
        if installed_dir.exists() {
            copy_dir_recursive(&installed_dir, &backup_path)?;
        }
        
        Ok(())
    }
    
    fn restore_backup(&self, name: &str) -> Result<()> {
        let backup_path = self.backup_dir.join(name);
        let installed_dir = self.root.join("var/lib/mix-pkg/installed").join(name);
        
        if backup_path.exists() {
            // Remove current
            let _ = fs::remove_dir_all(&installed_dir);
            
            // Restore backup
            copy_dir_recursive(&backup_path, &installed_dir)?;
        }
        
        Ok(())
    }
    
    fn run_hooks(&self) -> Result<()> {
        let hooks_dir = self.root.join("etc/mix-pkg/hooks.d");
        
        if !hooks_dir.exists() {
            return Ok(());
        }
        
        let mut hooks: Vec<_> = fs::read_dir(&hooks_dir)?
            .filter_map(|e| e.ok())
            .filter(|e| e.path().extension().map(|ext| ext == "hook").unwrap_or(false))
            .collect();
        
        hooks.sort_by_key(|e| e.file_name());
        
        for hook in hooks {
            let path = hook.path();
            if path.is_file() {
                std::process::Command::new("sh")
                    .arg(&path)
                    .env("TRANSACTION_ID", &self.id)
                    .env("TRANSACTION_STATE", format!("{:?}", self.state))
                    .status()
                    .context(format!("Failed to run hook: {:?}", path))?;
            }
        }
        
        Ok(())
    }
}

/// Transaction log for recovery
pub struct TransactionLog {
    log_dir: PathBuf,
}

impl TransactionLog {
    pub fn new(root: &Path) -> Self {
        Self {
            log_dir: root.join("var/lib/mix-pkg/transactions"),
        }
    }
    
    pub fn log_start(&self, txn: &Transaction) -> Result<()> {
        fs::create_dir_all(&self.log_dir)?;
        
        let log_file = self.log_dir.join(format!("{}.log", txn.id()));
        let mut content = format!("Transaction: {}\n", txn.id());
        content.push_str(&format!("Started: {}\n", chrono_timestamp()));
        content.push_str("Operations:\n");
        
        for op in txn.operations() {
            match op {
                TransactionOp::Install(pkg) => {
                    content.push_str(&format!("  INSTALL {} {}\n", pkg.name, pkg.version));
                }
                TransactionOp::Remove(name) => {
                    content.push_str(&format!("  REMOVE {}\n", name));
                }
                TransactionOp::Upgrade(old, new) => {
                    content.push_str(&format!("  UPGRADE {} {} -> {}\n", old.name, old.version, new.version));
                }
            }
        }
        
        fs::write(&log_file, content)?;
        Ok(())
    }
    
    pub fn log_complete(&self, txn: &Transaction) -> Result<()> {
        let log_file = self.log_dir.join(format!("{}.log", txn.id()));
        
        let mut content = fs::read_to_string(&log_file).unwrap_or_default();
        content.push_str(&format!("Completed: {}\n", chrono_timestamp()));
        content.push_str(&format!("State: {:?}\n", txn.state()));
        
        fs::write(&log_file, content)?;
        Ok(())
    }
    
    pub fn get_incomplete(&self) -> Result<Vec<String>> {
        let mut incomplete = Vec::new();
        
        if !self.log_dir.exists() {
            return Ok(incomplete);
        }
        
        for entry in fs::read_dir(&self.log_dir)? {
            let entry = entry?;
            let path = entry.path();
            
            if path.extension().map(|e| e == "log").unwrap_or(false) {
                let content = fs::read_to_string(&path)?;
                if !content.contains("Completed:") {
                    if let Some(name) = path.file_stem() {
                        incomplete.push(name.to_string_lossy().to_string());
                    }
                }
            }
        }
        
        Ok(incomplete)
    }
}

fn copy_dir_recursive(src: &Path, dst: &Path) -> Result<()> {
    fs::create_dir_all(dst)?;
    
    for entry in fs::read_dir(src)? {
        let entry = entry?;
        let src_path = entry.path();
        let dst_path = dst.join(entry.file_name());
        
        if src_path.is_dir() {
            copy_dir_recursive(&src_path, &dst_path)?;
        } else {
            fs::copy(&src_path, &dst_path)?;
        }
    }
    
    Ok(())
}

fn chrono_timestamp() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0)
}
