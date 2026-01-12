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

/// Create a sample training example (for testing)
pub fn create_sample_example() -> TrainingExample {
    TrainingExample {
        id: "sample_001".to_string(),
        category: "kernel_panic".to_string(),
        subcategory: Some("null_pointer".to_string()),
        input: ExampleInput {
            error: "Kernel panic - not syncing: Attempted to kill init!".to_string(),
            context: ExampleContext {
                kernel_version: Some("6.1.0-mixos".to_string()),
                boot_stage: "init".to_string(),
                last_action: Some("start_service".to_string()),
                uptime_seconds: Some(12),
            },
            state: ExampleState {
                memory_available: true,
                root_mounted: true,
                network_up: false,
                services_started: vec!["broker".to_string()],
            },
        },
        output: ExampleOutput {
            diagnosis: "Init process killed due to service failure".to_string(),
            root_cause: Some("Service dependency missing".to_string()),
            actions: vec![
                ExampleAction {
                    action: "disable_service".to_string(),
                    params: serde_json::json!({"service": "mix-agent", "temporary": true}),
                    order: 1,
                    fallback: Some("emergency_shell".to_string()),
                },
                ExampleAction {
                    action: "reboot".to_string(),
                    params: serde_json::json!({"mode": "normal"}),
                    order: 2,
                    fallback: None,
                },
            ],
            confidence: 0.92,
            severity: "critical".to_string(),
        },
        metadata: Some(ExampleMetadata {
            source: Some("synthetic".to_string()),
            verified: Some(true),
            added_date: Some("2026-01-12".to_string()),
            tags: Some(vec!["init".to_string(), "service".to_string()]),
        }),
    }
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
