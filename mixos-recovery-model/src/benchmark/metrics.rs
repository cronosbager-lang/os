//! Benchmark Metrics
//!
//! Defines all metrics for evaluating MRM performance.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Complete benchmark results
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BenchmarkResults {
    /// Model information
    pub model_info: ModelInfo,
    
    /// Overall metrics
    pub overall: OverallMetrics,
    
    /// Per-category metrics
    pub per_category: HashMap<String, CategoryMetrics>,
    
    /// Reasoning metrics
    pub reasoning: ReasoningMetrics,
    
    /// Action metrics
    pub action: ActionMetrics,
    
    /// Calibration metrics
    pub calibration: CalibrationMetrics,
    
    /// Performance metrics
    pub performance: PerformanceMetrics,
    
    /// Field dynamics metrics
    pub field_dynamics: FieldDynamicsMetrics,
}

/// Model information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModelInfo {
    pub name: String,
    pub version: String,
    pub variants: usize,
    pub domains: usize,
    pub model_size_bytes: usize,
    pub config: String,
}

/// Overall accuracy metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OverallMetrics {
    /// Overall accuracy (correct predictions / total)
    pub accuracy: f64,
    
    /// Macro F1 score (average F1 across categories)
    pub macro_f1: f64,
    
    /// Weighted F1 score (weighted by category frequency)
    pub weighted_f1: f64,
    
    /// Total examples evaluated
    pub total_examples: usize,
    
    /// Correct predictions
    pub correct: usize,
}

/// Per-category metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CategoryMetrics {
    pub category: String,
    pub accuracy: f64,
    pub precision: f64,
    pub recall: f64,
    pub f1: f64,
    pub support: usize,  // Number of examples
    
    /// Confusion with other categories
    pub confusion: HashMap<String, usize>,
}

/// Reasoning quality metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ReasoningMetrics {
    /// Correct root cause identification rate
    pub root_cause_accuracy: f64,
    
    /// Multi-step reasoning success (when multiple actions needed)
    pub multi_step_accuracy: f64,
    
    /// Causal chain correctness
    pub causal_accuracy: f64,
    
    /// Domain identification accuracy
    pub domain_accuracy: f64,
    
    /// Average reasoning depth (number of connected inferences)
    pub avg_reasoning_depth: f64,
    
    /// Reasoning coherence (internal consistency)
    pub coherence_score: f64,
}

/// Action prediction metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActionMetrics {
    /// Primary action accuracy (first action correct)
    pub primary_action_accuracy: f64,
    
    /// Full sequence accuracy (all actions in correct order)
    pub sequence_accuracy: f64,
    
    /// Action F1 score
    pub action_f1: f64,
    
    /// Parameter accuracy (correct params for correct actions)
    pub param_accuracy: f64,
    
    /// Fallback appropriateness (correct fallback when needed)
    pub fallback_accuracy: f64,
    
    /// Per-action metrics
    pub per_action: HashMap<String, ActionTypeMetrics>,
}

/// Metrics for a specific action type
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActionTypeMetrics {
    pub action: String,
    pub precision: f64,
    pub recall: f64,
    pub f1: f64,
    pub support: usize,
}

/// Confidence calibration metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CalibrationMetrics {
    /// Expected Calibration Error
    pub ece: f64,
    
    /// Maximum Calibration Error
    pub mce: f64,
    
    /// Average confidence
    pub avg_confidence: f64,
    
    /// Confidence when correct
    pub confidence_when_correct: f64,
    
    /// Confidence when incorrect
    pub confidence_when_incorrect: f64,
    
    /// Calibration bins (for reliability diagram)
    pub calibration_bins: Vec<CalibrationBin>,
    
    /// Overconfidence rate (high confidence but wrong)
    pub overconfidence_rate: f64,
    
    /// Underconfidence rate (low confidence but right)
    pub underconfidence_rate: f64,
}

/// Single calibration bin
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CalibrationBin {
    pub bin_start: f64,
    pub bin_end: f64,
    pub avg_confidence: f64,
    pub accuracy: f64,
    pub count: usize,
}

/// Performance/latency metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceMetrics {
    /// Average inference time (ms)
    pub avg_latency_ms: f64,
    
    /// P50 latency
    pub p50_latency_ms: f64,
    
    /// P95 latency
    pub p95_latency_ms: f64,
    
    /// P99 latency
    pub p99_latency_ms: f64,
    
    /// Max latency
    pub max_latency_ms: f64,
    
    /// Throughput (inferences per second)
    pub throughput: f64,
    
    /// Average settling steps
    pub avg_settling_steps: f64,
    
    /// Memory usage (bytes)
    pub memory_usage_bytes: usize,
}

/// Field dynamics specific metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FieldDynamicsMetrics {
    /// Average order parameter (Kuramoto sync)
    pub avg_order_parameter: f64,
    
    /// Order parameter when correct
    pub order_param_when_correct: f64,
    
    /// Order parameter when incorrect
    pub order_param_when_incorrect: f64,
    
    /// Average resonance strength
    pub avg_resonance_strength: f64,
    
    /// Average coherence
    pub avg_coherence: f64,
    
    /// Convergence rate (% that converged)
    pub convergence_rate: f64,
    
    /// Average energy at convergence
    pub avg_final_energy: f64,
    
    /// Domain activation patterns
    pub domain_activations: HashMap<String, f64>,
}

impl BenchmarkResults {
    /// Create empty results
    pub fn new(model_info: ModelInfo) -> Self {
        Self {
            model_info,
            overall: OverallMetrics {
                accuracy: 0.0,
                macro_f1: 0.0,
                weighted_f1: 0.0,
                total_examples: 0,
                correct: 0,
            },
            per_category: HashMap::new(),
            reasoning: ReasoningMetrics {
                root_cause_accuracy: 0.0,
                multi_step_accuracy: 0.0,
                causal_accuracy: 0.0,
                domain_accuracy: 0.0,
                avg_reasoning_depth: 0.0,
                coherence_score: 0.0,
            },
            action: ActionMetrics {
                primary_action_accuracy: 0.0,
                sequence_accuracy: 0.0,
                action_f1: 0.0,
                param_accuracy: 0.0,
                fallback_accuracy: 0.0,
                per_action: HashMap::new(),
            },
            calibration: CalibrationMetrics {
                ece: 0.0,
                mce: 0.0,
                avg_confidence: 0.0,
                confidence_when_correct: 0.0,
                confidence_when_incorrect: 0.0,
                calibration_bins: Vec::new(),
                overconfidence_rate: 0.0,
                underconfidence_rate: 0.0,
            },
            performance: PerformanceMetrics {
                avg_latency_ms: 0.0,
                p50_latency_ms: 0.0,
                p95_latency_ms: 0.0,
                p99_latency_ms: 0.0,
                max_latency_ms: 0.0,
                throughput: 0.0,
                avg_settling_steps: 0.0,
                memory_usage_bytes: 0,
            },
            field_dynamics: FieldDynamicsMetrics {
                avg_order_parameter: 0.0,
                order_param_when_correct: 0.0,
                order_param_when_incorrect: 0.0,
                avg_resonance_strength: 0.0,
                avg_coherence: 0.0,
                convergence_rate: 0.0,
                avg_final_energy: 0.0,
                domain_activations: HashMap::new(),
            },
        }
    }

    /// Check if meets minimum targets
    pub fn meets_minimum_targets(&self) -> bool {
        self.overall.accuracy >= 0.95
            && self.action.action_f1 >= 0.90
            && self.calibration.ece <= 0.10
    }

    /// Check if meets target goals
    pub fn meets_target_goals(&self) -> bool {
        self.overall.accuracy >= 0.97
            && self.action.action_f1 >= 0.95
            && self.calibration.ece <= 0.05
    }

    /// Get summary string
    pub fn summary(&self) -> String {
        format!(
            "Accuracy: {:.2}% | F1: {:.2}% | ECE: {:.4} | Latency: {:.1}ms",
            self.overall.accuracy * 100.0,
            self.action.action_f1 * 100.0,
            self.calibration.ece,
            self.performance.avg_latency_ms
        )
    }
}

/// Single evaluation result for one example
#[derive(Debug, Clone)]
pub struct EvalResult {
    pub example_id: String,
    pub category: String,
    pub correct: bool,
    pub primary_action_correct: bool,
    pub sequence_correct: bool,
    pub params_correct: bool,
    pub root_cause_correct: bool,
    pub confidence: f64,
    pub latency_ms: f64,
    pub order_parameter: f64,
    pub coherence: f64,
    pub settling_steps: usize,
    pub predicted_actions: Vec<String>,
    pub expected_actions: Vec<String>,
    pub predicted_domain: String,
    pub expected_domain: String,
}
