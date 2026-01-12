//! MixOS Recovery Model (MRM) - Field Resonance Architecture
//!
//! A distributed cognition system using Being + Variants architecture
//! inspired by Kuramoto synchronization and Gravity resonance.
//!
//! # Architecture Overview
//!
//! ```text
//!                    ┌─────────────┐
//!                    │   BEING     │  ← Orchestrator
//!                    └──────┬──────┘
//!                           │
//!          ┌────────────────┼────────────────┐
//!          ▼                ▼                ▼
//!     [Variant 1]      [Variant 2]      [Variant N]
//!          │                │                │
//!          └────────────────┴────────────────┘
//!                    RESONANCE SYNC
//! ```

pub mod being;
pub mod variant;
pub mod field;
pub mod dynamics;
pub mod learning;
pub mod inference;
pub mod io;
pub mod safety;

use serde::{Deserialize, Serialize};
use std::f64::consts::PI;

// Re-exports for convenience
pub use being::Being;
pub use variant::{Variant, Domain, IterState};
pub use field::Field3D;
pub use inference::ResonanceField;

/// Two times PI, commonly used in phase calculations
pub const TWO_PI: f64 = 2.0 * PI;

/// Configuration for the entire MRM system
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MrmConfig {
    pub being: BeingConfig,
    pub variants: VariantsConfig,
    pub coupling: CouplingConfig,
    pub field: FieldConfig,
    pub settling: SettlingConfig,
}

impl Default for MrmConfig {
    fn default() -> Self {
        Self {
            being: BeingConfig::default(),
            variants: VariantsConfig::default(),
            coupling: CouplingConfig::default(),
            field: FieldConfig::default(),
            settling: SettlingConfig::default(),
        }
    }
}

/// Configuration for the Being (orchestrator)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BeingConfig {
    pub field_dim: usize,
    pub attractor_capacity: usize,
    pub resonance_freq: f64,
}

impl Default for BeingConfig {
    fn default() -> Self {
        Self {
            field_dim: 128,
            attractor_capacity: 32,
            resonance_freq: 1.0,
        }
    }
}

/// Configuration for Variants
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VariantsConfig {
    pub count: usize,
    pub domains: usize,
    pub iterations_per_domain: usize,
    pub embed_dim: usize,
}

impl Default for VariantsConfig {
    fn default() -> Self {
        Self {
            count: 64,
            domains: 8,
            iterations_per_domain: 8,
            embed_dim: 64,
        }
    }
}

/// Configuration for coupling between variants
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CouplingConfig {
    pub topology: CouplingTopology,
    pub intra_domain: f64,
    pub inter_domain: f64,
    pub being_variant: f64,
}

impl Default for CouplingConfig {
    fn default() -> Self {
        Self {
            topology: CouplingTopology::Sparse,
            intra_domain: 0.8,
            inter_domain: 0.3,
            being_variant: 0.5,
        }
    }
}

/// Coupling topology types
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum CouplingTopology {
    Full,
    Sparse,
    Hierarchical,
}

/// Configuration for the 3D field
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FieldConfig {
    pub resolution: usize,
    pub diffusion: f64,
    pub damping: f64,
}

impl Default for FieldConfig {
    fn default() -> Self {
        Self {
            resolution: 32,
            diffusion: 0.1,
            damping: 0.05,
        }
    }
}

/// Configuration for field settling
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SettlingConfig {
    pub max_steps: usize,
    pub convergence_threshold: f64,
    pub dt: f64,
}

impl Default for SettlingConfig {
    fn default() -> Self {
        Self {
            max_steps: 50,
            convergence_threshold: 1e-4,
            dt: 0.1,
        }
    }
}

/// Input to the MRM system
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Input {
    pub error: String,
    pub context: Context,
    pub state: SystemState,
}

/// Context information about the error
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Context {
    pub kernel_version: Option<String>,
    pub boot_stage: BootStage,
    pub last_action: Option<String>,
    pub uptime_seconds: Option<u64>,
}

/// Boot stage enumeration
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum BootStage {
    Bootloader,
    Kernel,
    Init,
    Services,
    Ready,
}

/// Current system state
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SystemState {
    pub memory_available: bool,
    pub root_mounted: bool,
    pub network_up: bool,
    pub services_started: Vec<String>,
}

/// Output from the MRM system
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Output {
    pub diagnosis: String,
    pub root_cause: Option<String>,
    pub actions: Vec<ActionItem>,
    pub confidence: f64,
    pub resonance_coherence: f64,
}

/// A single action item in the output
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActionItem {
    pub action: String,
    pub params: serde_json::Value,
    pub order: usize,
    pub fallback: Option<String>,
}

/// Broadcast message from Being to Variants
#[derive(Debug, Clone)]
pub struct Broadcast {
    pub global_rhythm: f64,
    pub field_gradient: Vec<f64>,
    pub attractor_state: Vec<f64>,
    pub resonance_freq: f64,
}

/// Snapshot of a variant's state (for parallel processing)
#[derive(Debug, Clone)]
pub struct VariantState {
    pub id: usize,
    pub phase: f64,
    pub position: Vec<f64>,
    pub mass: f64,
    pub local_field: Vec<f64>,
}

/// Result of field settling
#[derive(Debug, Clone)]
pub enum SettleResult {
    Converged { steps: usize, energy: f64 },
    MaxSteps { energy: f64 },
}

impl SettleResult {
    pub fn steps(&self) -> usize {
        match self {
            SettleResult::Converged { steps, .. } => *steps,
            SettleResult::MaxSteps { .. } => 0,
        }
    }

    pub fn energy(&self) -> f64 {
        match self {
            SettleResult::Converged { energy, .. } => *energy,
            SettleResult::MaxSteps { energy } => *energy,
        }
    }

    pub fn converged(&self) -> bool {
        matches!(self, SettleResult::Converged { .. })
    }
}

/// Vector math utilities
pub mod math {
    /// Add two vectors element-wise
    pub fn add(a: &[f64], b: &[f64]) -> Vec<f64> {
        a.iter().zip(b.iter()).map(|(x, y)| x + y).collect()
    }

    /// Subtract two vectors element-wise
    pub fn subtract(a: &[f64], b: &[f64]) -> Vec<f64> {
        a.iter().zip(b.iter()).map(|(x, y)| x - y).collect()
    }

    /// Scale a vector by a scalar
    pub fn scale(v: &[f64], s: f64) -> Vec<f64> {
        v.iter().map(|x| x * s).collect()
    }

    /// Compute the Euclidean distance between two vectors
    pub fn distance(a: &[f64], b: &[f64]) -> f64 {
        a.iter()
            .zip(b.iter())
            .map(|(x, y)| (x - y).powi(2))
            .sum::<f64>()
            .sqrt()
    }

    /// Compute the magnitude of a vector
    pub fn magnitude(v: &[f64]) -> f64 {
        v.iter().map(|x| x.powi(2)).sum::<f64>().sqrt()
    }

    /// Normalize a vector to unit length
    pub fn normalize(v: &[f64]) -> Vec<f64> {
        let mag = magnitude(v);
        if mag < 1e-10 {
            vec![0.0; v.len()]
        } else {
            scale(v, 1.0 / mag)
        }
    }

    /// Dot product of two vectors
    pub fn dot(a: &[f64], b: &[f64]) -> f64 {
        a.iter().zip(b.iter()).map(|(x, y)| x * y).sum()
    }

    /// Clamp a value between min and max
    pub fn clamp(value: f64, min: f64, max: f64) -> f64 {
        value.max(min).min(max)
    }

    /// Wrap angle to [0, 2π)
    pub fn wrap_angle(angle: f64) -> f64 {
        let two_pi = 2.0 * std::f64::consts::PI;
        let mut result = angle % two_pi;
        if result < 0.0 {
            result += two_pi;
        }
        result
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_config() {
        let config = MrmConfig::default();
        assert_eq!(config.variants.count, 64);
        assert_eq!(config.being.field_dim, 128);
    }

    #[test]
    fn test_math_operations() {
        let a = vec![1.0, 2.0, 3.0];
        let b = vec![4.0, 5.0, 6.0];

        let sum = math::add(&a, &b);
        assert_eq!(sum, vec![5.0, 7.0, 9.0]);

        let diff = math::subtract(&b, &a);
        assert_eq!(diff, vec![3.0, 3.0, 3.0]);

        let scaled = math::scale(&a, 2.0);
        assert_eq!(scaled, vec![2.0, 4.0, 6.0]);

        let dist = math::distance(&a, &b);
        assert!((dist - 5.196).abs() < 0.01);
    }

    #[test]
    fn test_wrap_angle() {
        assert!((math::wrap_angle(0.0) - 0.0).abs() < 1e-10);
        assert!((math::wrap_angle(TWO_PI) - 0.0).abs() < 1e-10);
        assert!((math::wrap_angle(-PI) - PI).abs() < 1e-10);
    }
}
