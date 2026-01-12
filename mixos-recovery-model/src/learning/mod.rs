//! Learning Module
//!
//! Contains the four learning mechanisms:
//! - Hebbian: Coupling adjustment ("neurons that fire together, wire together")
//! - Resonance: Frequency tuning
//! - Attractor: Attractor sculpting

pub mod hebbian;
pub mod resonance;
pub mod attractor;

pub use hebbian::HebbianLearning;
pub use resonance::ResonanceTuning;
pub use attractor::AttractorSculpting;

use crate::{Output, VariantState};
use crate::variant::Variant;
use crate::being::Being;

/// Combined learning system
pub struct LearningSystem {
    pub hebbian: HebbianLearning,
    pub resonance: ResonanceTuning,
    pub attractor: AttractorSculpting,
}

impl LearningSystem {
    /// Create new learning system
    pub fn new(learning_rate: f64) -> Self {
        Self {
            hebbian: HebbianLearning::new(learning_rate),
            resonance: ResonanceTuning::new(learning_rate * 0.1),
            attractor: AttractorSculpting::new(learning_rate * 0.5),
        }
    }

    /// Perform a complete learning step
    pub fn learn(
        &self,
        being: &mut Being,
        variants: &mut [Variant],
        coupling_matrix: &mut Vec<Vec<f64>>,
        actual: &Output,
        expected: &Output,
    ) {
        // Compute error
        let error = compute_error(actual, expected);
        
        // 1. Hebbian coupling adjustment
        let states: Vec<VariantState> = variants.iter().map(|v| v.state()).collect();
        self.hebbian.update_coupling(coupling_matrix, &states, error);
        
        // 2. Resonance tuning
        self.resonance.tune_frequencies(variants, actual, error);
        
        // 3. Attractor sculpting
        self.attractor.sculpt(being, actual, expected, error);
    }
}

/// Compute error between actual and expected output
pub fn compute_error(actual: &Output, expected: &Output) -> f64 {
    if actual.actions.is_empty() && expected.actions.is_empty() {
        return 0.0;
    }
    
    if actual.actions.is_empty() || expected.actions.is_empty() {
        return 1.0;
    }
    
    // Compare action sequences
    let mut matches = 0;
    let total = expected.actions.len().max(actual.actions.len());
    
    for (a, e) in actual.actions.iter().zip(expected.actions.iter()) {
        if a.action == e.action {
            matches += 1;
            
            // Bonus for matching parameters
            if a.params == e.params {
                matches += 1;
            }
        }
    }
    
    let max_possible = total * 2; // action + params
    1.0 - (matches as f64 / max_possible as f64)
}

/// Learning configuration
#[derive(Debug, Clone)]
pub struct LearningConfig {
    pub learning_rate: f64,
    pub coupling_decay: f64,
    pub mass_decay: f64,
    pub momentum: f64,
}

impl Default for LearningConfig {
    fn default() -> Self {
        Self {
            learning_rate: 0.01,
            coupling_decay: 0.001,
            mass_decay: 0.01,
            momentum: 0.9,
        }
    }
}
