//! I/O Module
//!
//! Handles model serialization (GGUF format) and dataset loading.

pub mod gguf;
pub mod dataset;

pub use gguf::{GgufWriter, GgufReader};
pub use dataset::{DatasetLoader, TrainingExample};
