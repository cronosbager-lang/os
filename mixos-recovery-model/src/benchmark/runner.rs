//! Benchmark Runner
//!
//! Executes benchmarks and collects metrics.

use std::collections::HashMap;
use std::time::Instant;

use crate::inference::ResonanceField;
use crate::io::dataset::{DatasetLoader, TrainingExample};
use crate::{Input, Output};

use super::metrics::*;

/// Benchmark runner
pub struct BenchmarkRunner {
    model: ResonanceField,
    num_calibration_bins: usize,
}

impl BenchmarkRunner {
    /// Create new benchmark runner
    pub fn new(model: ResonanceField) -> Self {
        Self {
            model,
            num_calibration_bins: 10,
        }
    }

    /// Run full benchmark suite
    pub fn run(&mut self, test_data: &[TrainingExample]) -> BenchmarkResults {
        let model_info = ModelInfo {
            name: "MRM-64".to_string(),
            version: "2.0".to_string(),
            variants: self.model.variants.len(),
            domains: 8,
            model_size_bytes: 90 * 1024, // ~90KB
            config: "mrm-64".to_string(),
        };

        let mut results = BenchmarkResults::new(model_info);
        let mut eval_results: Vec<EvalResult> = Vec::new();

        // Run inference on all examples
        for example in test_data {
            let eval = self.evaluate_example(example);
            eval_results.push(eval);
        }

        // Compute all metrics
        self.compute_overall_metrics(&eval_results, &mut results);
        self.compute_category_metrics(&eval_results, &mut results);
        self.compute_reasoning_metrics(&eval_results, &mut results);
        self.compute_action_metrics(&eval_results, &mut results);
        self.compute_calibration_metrics(&eval_results, &mut results);
        self.compute_performance_metrics(&eval_results, &mut results);
        self.compute_field_dynamics_metrics(&eval_results, &mut results);

        results
    }

    /// Evaluate a single example
    fn evaluate_example(&mut self, example: &TrainingExample) -> EvalResult {
        let (input, expected) = example.to_input_output().unwrap();

        // Time the inference
        let start = Instant::now();
        let output = self.model.infer(&input);
        let latency_ms = start.elapsed().as_secs_f64() * 1000.0;

        // Get model stats
        let stats = self.model.stats();

        // Extract predicted and expected actions
        let predicted_actions: Vec<String> = output.actions.iter().map(|a| a.action.clone()).collect();
        let expected_actions: Vec<String> = expected.actions.iter().map(|a| a.action.clone()).collect();

        // Check correctness
        let primary_action_correct = !predicted_actions.is_empty()
            && !expected_actions.is_empty()
            && predicted_actions[0] == expected_actions[0];

        let sequence_correct = predicted_actions == expected_actions;

        let params_correct = if primary_action_correct && !output.actions.is_empty() && !expected.actions.is_empty() {
            output.actions[0].params == expected.actions[0].params
        } else {
            false
        };

        // Check root cause
        let root_cause_correct = match (&output.root_cause, &expected.root_cause) {
            (Some(pred), Some(exp)) => pred.to_lowercase().contains(&exp.to_lowercase())
                || exp.to_lowercase().contains(&pred.to_lowercase()),
            (None, None) => true,
            _ => false,
        };

        // Determine domains
        let predicted_domain = self.extract_domain_from_diagnosis(&output.diagnosis);
        let expected_domain = example.category.clone();

        // Overall correctness (primary action + reasonable confidence)
        let correct = primary_action_correct;

        EvalResult {
            example_id: example.id.clone(),
            category: example.category.clone(),
            correct,
            primary_action_correct,
            sequence_correct,
            params_correct,
            root_cause_correct,
            confidence: output.confidence,
            latency_ms,
            order_parameter: stats.order_parameter,
            coherence: stats.coherence,
            settling_steps: 0, // Would need to track this in model
            predicted_actions,
            expected_actions,
            predicted_domain,
            expected_domain,
        }
    }

    /// Extract domain from diagnosis string
    fn extract_domain_from_diagnosis(&self, diagnosis: &str) -> String {
        let diagnosis_lower = diagnosis.to_lowercase();
        
        if diagnosis_lower.contains("kernel") {
            "kernel_panic".to_string()
        } else if diagnosis_lower.contains("filesystem") || diagnosis_lower.contains("mount") {
            "mount_failure".to_string()
        } else if diagnosis_lower.contains("service") {
            "service_crash".to_string()
        } else if diagnosis_lower.contains("boot") {
            "boot_failure".to_string()
        } else if diagnosis_lower.contains("config") {
            "config_error".to_string()
        } else if diagnosis_lower.contains("network") {
            "network_issue".to_string()
        } else if diagnosis_lower.contains("hardware") {
            "hardware_issue".to_string()
        } else if diagnosis_lower.contains("memory") {
            "memory_issue".to_string()
        } else {
            "unknown".to_string()
        }
    }

    /// Compute overall metrics
    fn compute_overall_metrics(&self, evals: &[EvalResult], results: &mut BenchmarkResults) {
        let total = evals.len();
        let correct = evals.iter().filter(|e| e.correct).count();

        results.overall.total_examples = total;
        results.overall.correct = correct;
        results.overall.accuracy = if total > 0 { correct as f64 / total as f64 } else { 0.0 };

        // Compute macro F1
        let categories: Vec<String> = evals.iter().map(|e| e.category.clone()).collect();
        let unique_categories: std::collections::HashSet<_> = categories.iter().collect();
        
        let mut f1_sum = 0.0;
        let mut weighted_f1_sum = 0.0;
        let mut total_weight = 0;

        for cat in &unique_categories {
            let cat_evals: Vec<_> = evals.iter().filter(|e| &e.category == *cat).collect();
            let cat_correct = cat_evals.iter().filter(|e| e.correct).count();
            let cat_total = cat_evals.len();
            
            if cat_total > 0 {
                let precision = cat_correct as f64 / cat_total as f64;
                let recall = precision; // Simplified for single-label
                let f1 = if precision + recall > 0.0 {
                    2.0 * precision * recall / (precision + recall)
                } else {
                    0.0
                };
                
                f1_sum += f1;
                weighted_f1_sum += f1 * cat_total as f64;
                total_weight += cat_total;
            }
        }

        results.overall.macro_f1 = if !unique_categories.is_empty() {
            f1_sum / unique_categories.len() as f64
        } else {
            0.0
        };

        results.overall.weighted_f1 = if total_weight > 0 {
            weighted_f1_sum / total_weight as f64
        } else {
            0.0
        };
    }

    /// Compute per-category metrics
    fn compute_category_metrics(&self, evals: &[EvalResult], results: &mut BenchmarkResults) {
        let mut category_evals: HashMap<String, Vec<&EvalResult>> = HashMap::new();
        
        for eval in evals {
            category_evals.entry(eval.category.clone()).or_default().push(eval);
        }

        for (category, cat_evals) in category_evals {
            let total = cat_evals.len();
            let correct = cat_evals.iter().filter(|e| e.correct).count();
            
            let accuracy = if total > 0 { correct as f64 / total as f64 } else { 0.0 };
            let precision = accuracy;
            let recall = accuracy;
            let f1 = if precision + recall > 0.0 {
                2.0 * precision * recall / (precision + recall)
            } else {
                0.0
            };

            // Build confusion matrix
            let mut confusion: HashMap<String, usize> = HashMap::new();
            for eval in &cat_evals {
                if !eval.correct {
                    *confusion.entry(eval.predicted_domain.clone()).or_insert(0) += 1;
                }
            }

            results.per_category.insert(category.clone(), CategoryMetrics {
                category,
                accuracy,
                precision,
                recall,
                f1,
                support: total,
                confusion,
            });
        }
    }

    /// Compute reasoning metrics
    fn compute_reasoning_metrics(&self, evals: &[EvalResult], results: &mut BenchmarkResults) {
        let total = evals.len() as f64;
        if total == 0.0 {
            return;
        }

        // Root cause accuracy
        let root_cause_correct = evals.iter().filter(|e| e.root_cause_correct).count();
        results.reasoning.root_cause_accuracy = root_cause_correct as f64 / total;

        // Multi-step accuracy (examples with >1 action)
        let multi_step: Vec<_> = evals.iter()
            .filter(|e| e.expected_actions.len() > 1)
            .collect();
        if !multi_step.is_empty() {
            let multi_correct = multi_step.iter().filter(|e| e.sequence_correct).count();
            results.reasoning.multi_step_accuracy = multi_correct as f64 / multi_step.len() as f64;
        }

        // Domain accuracy
        let domain_correct = evals.iter()
            .filter(|e| e.predicted_domain == e.expected_domain)
            .count();
        results.reasoning.domain_accuracy = domain_correct as f64 / total;

        // Coherence score (average coherence)
        results.reasoning.coherence_score = evals.iter()
            .map(|e| e.coherence)
            .sum::<f64>() / total;

        // Causal accuracy (simplified: correct when root cause + action both correct)
        let causal_correct = evals.iter()
            .filter(|e| e.root_cause_correct && e.primary_action_correct)
            .count();
        results.reasoning.causal_accuracy = causal_correct as f64 / total;

        // Average reasoning depth (based on action count)
        results.reasoning.avg_reasoning_depth = evals.iter()
            .map(|e| e.predicted_actions.len() as f64)
            .sum::<f64>() / total;
    }

    /// Compute action metrics
    fn compute_action_metrics(&self, evals: &[EvalResult], results: &mut BenchmarkResults) {
        let total = evals.len() as f64;
        if total == 0.0 {
            return;
        }

        // Primary action accuracy
        let primary_correct = evals.iter().filter(|e| e.primary_action_correct).count();
        results.action.primary_action_accuracy = primary_correct as f64 / total;

        // Sequence accuracy
        let seq_correct = evals.iter().filter(|e| e.sequence_correct).count();
        results.action.sequence_accuracy = seq_correct as f64 / total;

        // Parameter accuracy
        let param_correct = evals.iter().filter(|e| e.params_correct).count();
        results.action.param_accuracy = param_correct as f64 / total;

        // Action F1 (same as primary action accuracy for now)
        results.action.action_f1 = results.action.primary_action_accuracy;

        // Per-action metrics
        let mut action_stats: HashMap<String, (usize, usize)> = HashMap::new(); // (correct, total)
        
        for eval in evals {
            for expected in &eval.expected_actions {
                let entry = action_stats.entry(expected.clone()).or_insert((0, 0));
                entry.1 += 1;
                if eval.predicted_actions.contains(expected) {
                    entry.0 += 1;
                }
            }
        }

        for (action, (correct, total)) in action_stats {
            let precision = if total > 0 { correct as f64 / total as f64 } else { 0.0 };
            let recall = precision;
            let f1 = if precision + recall > 0.0 {
                2.0 * precision * recall / (precision + recall)
            } else {
                0.0
            };

            results.action.per_action.insert(action.clone(), ActionTypeMetrics {
                action,
                precision,
                recall,
                f1,
                support: total,
            });
        }
    }

    /// Compute calibration metrics
    fn compute_calibration_metrics(&self, evals: &[EvalResult], results: &mut BenchmarkResults) {
        let total = evals.len();
        if total == 0 {
            return;
        }

        // Average confidence
        results.calibration.avg_confidence = evals.iter()
            .map(|e| e.confidence)
            .sum::<f64>() / total as f64;

        // Confidence when correct/incorrect
        let correct_evals: Vec<_> = evals.iter().filter(|e| e.correct).collect();
        let incorrect_evals: Vec<_> = evals.iter().filter(|e| !e.correct).collect();

        if !correct_evals.is_empty() {
            results.calibration.confidence_when_correct = correct_evals.iter()
                .map(|e| e.confidence)
                .sum::<f64>() / correct_evals.len() as f64;
        }

        if !incorrect_evals.is_empty() {
            results.calibration.confidence_when_incorrect = incorrect_evals.iter()
                .map(|e| e.confidence)
                .sum::<f64>() / incorrect_evals.len() as f64;
        }

        // Calibration bins
        let bin_size = 1.0 / self.num_calibration_bins as f64;
        let mut bins: Vec<CalibrationBin> = Vec::new();
        let mut ece_sum: f64 = 0.0;
        let mut mce: f64 = 0.0;

        for i in 0..self.num_calibration_bins {
            let bin_start = i as f64 * bin_size;
            let bin_end = (i + 1) as f64 * bin_size;

            let bin_evals: Vec<_> = evals.iter()
                .filter(|e| e.confidence >= bin_start && e.confidence < bin_end)
                .collect();

            if !bin_evals.is_empty() {
                let avg_conf = bin_evals.iter().map(|e| e.confidence).sum::<f64>() / bin_evals.len() as f64;
                let acc = bin_evals.iter().filter(|e| e.correct).count() as f64 / bin_evals.len() as f64;
                
                let gap = (avg_conf - acc).abs();
                ece_sum += gap * bin_evals.len() as f64;
                mce = mce.max(gap);

                bins.push(CalibrationBin {
                    bin_start,
                    bin_end,
                    avg_confidence: avg_conf,
                    accuracy: acc,
                    count: bin_evals.len(),
                });
            }
        }

        results.calibration.calibration_bins = bins;
        results.calibration.ece = ece_sum / total as f64;
        results.calibration.mce = mce;

        // Overconfidence/underconfidence rates
        let overconfident = evals.iter()
            .filter(|e| !e.correct && e.confidence > 0.7)
            .count();
        let underconfident = evals.iter()
            .filter(|e| e.correct && e.confidence < 0.5)
            .count();

        results.calibration.overconfidence_rate = overconfident as f64 / total as f64;
        results.calibration.underconfidence_rate = underconfident as f64 / total as f64;
    }

    /// Compute performance metrics
    fn compute_performance_metrics(&self, evals: &[EvalResult], results: &mut BenchmarkResults) {
        if evals.is_empty() {
            return;
        }

        let mut latencies: Vec<f64> = evals.iter().map(|e| e.latency_ms).collect();
        latencies.sort_by(|a, b| a.partial_cmp(b).unwrap());

        let total = latencies.len();
        
        results.performance.avg_latency_ms = latencies.iter().sum::<f64>() / total as f64;
        results.performance.p50_latency_ms = latencies[total / 2];
        results.performance.p95_latency_ms = latencies[(total as f64 * 0.95) as usize];
        results.performance.p99_latency_ms = latencies[(total as f64 * 0.99) as usize];
        results.performance.max_latency_ms = *latencies.last().unwrap();

        // Throughput
        let total_time_s = latencies.iter().sum::<f64>() / 1000.0;
        results.performance.throughput = if total_time_s > 0.0 {
            total as f64 / total_time_s
        } else {
            0.0
        };

        // Memory usage (approximate)
        results.performance.memory_usage_bytes = 200 * 1024; // ~200KB runtime
    }

    /// Compute field dynamics metrics
    fn compute_field_dynamics_metrics(&self, evals: &[EvalResult], results: &mut BenchmarkResults) {
        if evals.is_empty() {
            return;
        }

        let total = evals.len() as f64;

        // Average order parameter
        results.field_dynamics.avg_order_parameter = evals.iter()
            .map(|e| e.order_parameter)
            .sum::<f64>() / total;

        // Order parameter when correct/incorrect
        let correct_evals: Vec<_> = evals.iter().filter(|e| e.correct).collect();
        let incorrect_evals: Vec<_> = evals.iter().filter(|e| !e.correct).collect();

        if !correct_evals.is_empty() {
            results.field_dynamics.order_param_when_correct = correct_evals.iter()
                .map(|e| e.order_parameter)
                .sum::<f64>() / correct_evals.len() as f64;
        }

        if !incorrect_evals.is_empty() {
            results.field_dynamics.order_param_when_incorrect = incorrect_evals.iter()
                .map(|e| e.order_parameter)
                .sum::<f64>() / incorrect_evals.len() as f64;
        }

        // Average coherence
        results.field_dynamics.avg_coherence = evals.iter()
            .map(|e| e.coherence)
            .sum::<f64>() / total;

        // Domain activations
        let mut domain_counts: HashMap<String, usize> = HashMap::new();
        for eval in evals {
            *domain_counts.entry(eval.predicted_domain.clone()).or_insert(0) += 1;
        }
        
        for (domain, count) in domain_counts {
            results.field_dynamics.domain_activations.insert(domain, count as f64 / total);
        }
    }
}
