//! Dataset Loading
//!
//! Load training data from JSONL files.

use std::fs::File;
use std::io::{BufRead, BufReader};
use std::path::Path;

use serde::{Deserialize, Serialize};
use thiserror::Error;

use crate::{Input, Output, Context, SystemState, BootStage, ActionItem};

/// Dataset errors
#[derive(Error, Debug)]
pub enum DatasetError {
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),
    
    #[error("JSON parse error: {0}")]
    Json(#[from] serde_json::Error),
    
    #[error("Invalid example: {0}")]
    InvalidExample(String),
}

/// A single training example
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TrainingExample {
    pub id: String,
    pub category: String,
    pub subcategory: Option<String>,
    pub input: ExampleInput,
    pub output: ExampleOutput,
    pub metadata: Option<ExampleMetadata>,
}

/// Input portion of a training example
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExampleInput {
    pub error: String,
    pub context: ExampleContext,
    pub state: ExampleState,
}

/// Context in training example
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExampleContext {
    pub kernel_version: Option<String>,
    pub boot_stage: String,
    pub last_action: Option<String>,
    pub uptime_seconds: Option<u64>,
}

/// State in training example
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExampleState {
    pub memory_available: bool,
    pub root_mounted: bool,
    pub network_up: bool,
    pub services_started: Vec<String>,
}

/// Output portion of a training example
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExampleOutput {
    pub diagnosis: String,
    pub root_cause: Option<String>,
    pub actions: Vec<ExampleAction>,
    pub confidence: f64,
    pub severity: String,
}

/// Action in training example
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExampleAction {
    pub action: String,
    pub params: serde_json::Value,
    pub order: usize,
    pub fallback: Option<String>,
}

/// Metadata for training example
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExampleMetadata {
    pub source: Option<String>,
    pub verified: Option<bool>,
    pub added_date: Option<String>,
    pub tags: Option<Vec<String>>,
}

impl TrainingExample {
    /// Convert to Input/Output pair for training
    pub fn to_input_output(&self) -> Result<(Input, Output), DatasetError> {
        let boot_stage = match self.input.context.boot_stage.to_lowercase().as_str() {
            "bootloader" => BootStage::Bootloader,
            "kernel" => BootStage::Kernel,
            "init" => BootStage::Init,
            "services" => BootStage::Services,
            "ready" => BootStage::Ready,
            _ => BootStage::Init,
        };
        
        let input = Input {
            error: self.input.error.clone(),
            context: Context {
                kernel_version: self.input.context.kernel_version.clone(),
                boot_stage,
                last_action: self.input.context.last_action.clone(),
                uptime_seconds: self.input.context.uptime_seconds,
            },
            state: SystemState {
                memory_available: self.input.state.memory_available,
                root_mounted: self.input.state.root_mounted,
                network_up: self.input.state.network_up,
                services_started: self.input.state.services_started.clone(),
            },
        };
        
        let output = Output {
            diagnosis: self.output.diagnosis.clone(),
            root_cause: self.output.root_cause.clone(),
            actions: self.output.actions.iter().map(|a| ActionItem {
                action: a.action.clone(),
                params: a.params.clone(),
                order: a.order,
                fallback: a.fallback.clone(),
            }).collect(),
            confidence: self.output.confidence,
            resonance_coherence: 0.0, // Will be computed during inference
        };
        
        Ok((input, output))
    }
}

/// Dataset loader
pub struct DatasetLoader {
    /// Training examples
    pub examples: Vec<TrainingExample>,
}

impl DatasetLoader {
    /// Create a new empty dataset loader
    pub fn new() -> Self {
        Self {
            examples: Vec::new(),
        }
    }

    /// Load examples from a JSONL file
    pub fn load_jsonl<P: AsRef<Path>>(&mut self, path: P) -> Result<usize, DatasetError> {
        let file = File::open(path)?;
        let reader = BufReader::new(file);
        
        let mut count = 0;
        for line in reader.lines() {
            let line = line?;
            if line.trim().is_empty() {
                continue;
            }
            
            let example: TrainingExample = serde_json::from_str(&line)?;
            self.examples.push(example);
            count += 1;
        }
        
        Ok(count)
    }

    /// Load examples from multiple JSONL files
    pub fn load_multiple<P: AsRef<Path>>(&mut self, paths: &[P]) -> Result<usize, DatasetError> {
        let mut total = 0;
        for path in paths {
            total += self.load_jsonl(path)?;
        }
        Ok(total)
    }

    /// Get all examples
    pub fn examples(&self) -> &[TrainingExample] {
        &self.examples
    }

    /// Get number of examples
    pub fn len(&self) -> usize {
        self.examples.len()
    }

    /// Check if empty
    pub fn is_empty(&self) -> bool {
        self.examples.is_empty()
    }

    /// Filter examples by category
    pub fn filter_by_category(&self, category: &str) -> Vec<&TrainingExample> {
        self.examples.iter()
            .filter(|e| e.category == category)
            .collect()
    }

    /// Filter examples by subcategory
    pub fn filter_by_subcategory(&self, subcategory: &str) -> Vec<&TrainingExample> {
        self.examples.iter()
            .filter(|e| e.subcategory.as_deref() == Some(subcategory))
            .collect()
    }

    /// Get category statistics
    pub fn category_stats(&self) -> std::collections::HashMap<String, usize> {
        let mut stats = std::collections::HashMap::new();
        for example in &self.examples {
            *stats.entry(example.category.clone()).or_insert(0) += 1;
        }
        stats
    }

    /// Shuffle examples
    pub fn shuffle(&mut self) {
        use rand::seq::SliceRandom;
        let mut rng = rand::thread_rng();
        self.examples.shuffle(&mut rng);
    }

    /// Split into train/validation sets
    pub fn split(&self, train_ratio: f64) -> (Vec<&TrainingExample>, Vec<&TrainingExample>) {
        let split_idx = (self.examples.len() as f64 * train_ratio) as usize;
        
        let train: Vec<_> = self.examples.iter().take(split_idx).collect();
        let valid: Vec<_> = self.examples.iter().skip(split_idx).collect();
        
        (train, valid)
    }

    /// Get batches for training
    pub fn batches(&self, batch_size: usize) -> Vec<Vec<&TrainingExample>> {
        self.examples.chunks(batch_size)
            .map(|chunk| chunk.iter().collect())
            .collect()
    }

    /// Convert all examples to Input/Output pairs
    pub fn to_input_output_pairs(&self) -> Result<Vec<(Input, Output)>, DatasetError> {
        self.examples.iter()
            .map(|e| e.to_input_output())
            .collect()
    }
}

impl Default for DatasetLoader {
    fn default() -> Self {
        Self::new()
    }
}

/// Sample examples for different categories
const SAMPLE_EXAMPLES: &[(&str, &str, &str, &str, &str, &str)] = &[
    // (category, subcategory, error, boot_stage, diagnosis, action)
    ("kernel_panic", "vfs_mount", "Kernel panic - not syncing: VFS: Unable to mount root fs on unknown-block(0,0)", "kernel", "Root filesystem mount failure", "emergency_shell"),
    ("kernel_panic", "init_killed", "Kernel panic - not syncing: Attempted to kill init!", "init", "Init process killed", "disable_service"),
    ("kernel_panic", "null_pointer", "BUG: kernel NULL pointer dereference at 0000000000000000", "kernel", "Null pointer dereference in kernel", "reboot"),
    ("mount_failure", "device_not_found", "mount: /dev/sda1: special device does not exist", "init", "Block device not found", "wait_and_retry"),
    ("mount_failure", "filesystem_corrupted", "EXT4-fs error (device sda1): ext4_lookup: deleted inode referenced", "init", "Filesystem corruption detected", "fsck"),
    ("mount_failure", "wrong_fstype", "mount: wrong fs type, bad option, bad superblock", "init", "Incorrect filesystem type", "remount_filesystem"),
    ("service_crash", "dependency_missing", "systemd[1]: mix-agent.service: Failed with result 'dependency'", "services", "Service dependency not met", "restart_service"),
    ("service_crash", "config_invalid", "nginx: [emerg] unknown directive in /etc/nginx/nginx.conf:10", "services", "Invalid service configuration", "restore_config"),
    ("service_crash", "port_in_use", "Error: listen EADDRINUSE: address already in use :::8080", "services", "Port already in use", "restart_service"),
    ("boot_failure", "grub_error", "error: file '/boot/vmlinuz' not found", "bootloader", "Kernel image not found", "emergency_shell"),
    ("boot_failure", "initramfs_missing", "Failed to execute /init (error -2)", "kernel", "Initramfs init not found", "emergency_shell"),
    ("config_error", "syntax_error", "YAML parse error: mapping values are not allowed here", "services", "Configuration syntax error", "restore_config"),
    ("config_error", "invalid_value", "Error: invalid value for 'timeout': expected integer", "services", "Invalid configuration value", "restore_config"),
    ("network_issue", "interface_down", "RTNETLINK answers: Network is unreachable", "services", "Network interface down", "network_reset"),
    ("network_issue", "dns_failure", "Temporary failure in name resolution", "services", "DNS resolution failed", "network_reset"),
    ("hardware_issue", "disk_failure", "ata1.00: failed command: READ FPDMA QUEUED", "kernel", "Disk read failure", "emergency_shell"),
    ("hardware_issue", "memory_error", "EDAC MC0: 1 CE memory read error", "kernel", "Memory error detected", "log_and_continue"),
    ("memory_issue", "oom_killer", "Out of memory: Killed process 1234 (java)", "services", "Process killed by OOM killer", "clear_cache"),
    ("memory_issue", "swap_exhausted", "swap_free: Bad swap file entry", "services", "Swap space exhausted", "clear_cache"),
];

/// Create a sample training example (for testing)
pub fn create_sample_example() -> TrainingExample {
    use rand::Rng;
    let mut rng = rand::thread_rng();
    let idx = rng.gen_range(0..SAMPLE_EXAMPLES.len());
    create_sample_example_by_index(idx)
}

/// Create a specific sample example by index
pub fn create_sample_example_by_index(idx: usize) -> TrainingExample {
    let (category, subcategory, error, boot_stage, diagnosis, action) = 
        SAMPLE_EXAMPLES[idx % SAMPLE_EXAMPLES.len()];
    
    TrainingExample {
        id: format!("sample_{:04}", idx),
        category: category.to_string(),
        subcategory: Some(subcategory.to_string()),
        input: ExampleInput {
            error: error.to_string(),
            context: ExampleContext {
                kernel_version: Some("6.1.0-mixos".to_string()),
                boot_stage: boot_stage.to_string(),
                last_action: None,
                uptime_seconds: Some(10 + (idx as u64 % 100)),
            },
            state: ExampleState {
                memory_available: !category.contains("memory"),
                root_mounted: !category.contains("mount") && boot_stage != "kernel",
                network_up: !category.contains("network") && boot_stage == "services",
                services_started: if boot_stage == "services" {
                    vec!["broker".to_string()]
                } else {
                    vec![]
                },
            },
        },
        output: ExampleOutput {
            diagnosis: diagnosis.to_string(),
            root_cause: Some(format!("{} issue in {} stage", category, boot_stage)),
            actions: vec![
                ExampleAction {
                    action: action.to_string(),
                    params: match action {
                        "reboot" => serde_json::json!({"mode": "normal"}),
                        "fsck" => serde_json::json!({"device": "/dev/sda1", "auto_fix": true}),
                        "restart_service" => serde_json::json!({"service": "mix-agent"}),
                        "disable_service" => serde_json::json!({"service": "mix-agent", "temporary": true}),
                        "restore_config" => serde_json::json!({"config_path": "/etc/mixos/config.yaml"}),
                        "network_reset" => serde_json::json!({"interface": "eth0"}),
                        "clear_cache" => serde_json::json!({"cache_type": "all"}),
                        "emergency_shell" => serde_json::json!({"message": "Manual intervention required"}),
                        "remount_filesystem" => serde_json::json!({"path": "/", "options": "rw"}),
                        "wait_and_retry" => serde_json::json!({"condition": "device_ready", "timeout_seconds": 30, "retry_action": "mount"}),
                        "log_and_continue" => serde_json::json!({"severity": "warning", "message": "Non-critical error"}),
                        _ => serde_json::json!({}),
                    },
                    order: 1,
                    fallback: Some("emergency_shell".to_string()),
                },
            ],
            confidence: 0.85 + (idx as f64 % 10.0) / 100.0,
            severity: if category.contains("panic") || category.contains("boot") {
                "critical".to_string()
            } else if category.contains("failure") {
                "error".to_string()
            } else {
                "warning".to_string()
            },
        },
        metadata: Some(ExampleMetadata {
            source: Some("synthetic".to_string()),
            verified: Some(true),
            added_date: Some("2026-01-13".to_string()),
            tags: Some(vec![category.to_string(), boot_stage.to_string()]),
        }),
    }
}

/// Generate multiple diverse sample examples
pub fn generate_sample_examples(count: usize) -> Vec<TrainingExample> {
    (0..count).map(|i| create_sample_example_by_index(i)).collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;
    use tempfile::NamedTempFile;

    #[test]
    fn test_sample_example() {
        let example = create_sample_example();
        
        assert_eq!(example.category, "kernel_panic");
        assert_eq!(example.output.actions.len(), 2);
    }

    #[test]
    fn test_to_input_output() {
        let example = create_sample_example();
        let (input, output) = example.to_input_output().unwrap();
        
        assert!(input.error.contains("Kernel panic"));
        assert_eq!(output.actions.len(), 2);
    }

    #[test]
    fn test_load_jsonl() {
        let example = create_sample_example();
        let json = serde_json::to_string(&example).unwrap();
        
        let mut file = NamedTempFile::new().unwrap();
        writeln!(file, "{}", json).unwrap();
        writeln!(file, "{}", json).unwrap();
        
        let mut loader = DatasetLoader::new();
        let count = loader.load_jsonl(file.path()).unwrap();
        
        assert_eq!(count, 2);
        assert_eq!(loader.len(), 2);
    }

    #[test]
    fn test_category_stats() {
        let mut loader = DatasetLoader::new();
        
        let mut example1 = create_sample_example();
        example1.category = "kernel_panic".to_string();
        
        let mut example2 = create_sample_example();
        example2.category = "mount_failure".to_string();
        
        loader.examples.push(example1);
        loader.examples.push(example2.clone());
        loader.examples.push(example2);
        
        let stats = loader.category_stats();
        
        assert_eq!(stats.get("kernel_panic"), Some(&1));
        assert_eq!(stats.get("mount_failure"), Some(&2));
    }

    #[test]
    fn test_split() {
        let mut loader = DatasetLoader::new();
        
        for i in 0..10 {
            let mut example = create_sample_example();
            example.id = format!("example_{}", i);
            loader.examples.push(example);
        }
        
        let (train, valid) = loader.split(0.8);
        
        assert_eq!(train.len(), 8);
        assert_eq!(valid.len(), 2);
    }
}
