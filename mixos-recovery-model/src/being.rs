//! Being - The Central Orchestrator
//!
//! Being is the "conductor" of the field resonance system.
//! It broadcasts global rhythm to all variants and integrates their contributions.

use crate::{
    math, BeingConfig, Broadcast, FieldConfig, Output, ActionItem, TWO_PI,
};
use crate::field::Field3D;
use crate::variant::Variant;
use serde::{Deserialize, Serialize};

/// Being - The Central Orchestrator/Conductor
///
/// Responsibilities:
/// - Broadcast global rhythm to all variants
/// - Maintain field topology (solution space)
/// - Integrate variant contributions
/// - Compute overall confidence
/// - Decode final output from field state
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Being {
    /// Global clock/rhythm phase
    pub master_phase: f64,
    
    /// Master field structure (3D topology)
    #[serde(skip)]
    pub field_topology: Field3D,
    
    /// Fundamental resonance frequency
    pub resonance_freq: f64,
    
    /// Solution landscape - attractor states
    pub attractor_state: Vec<f64>,
    
    /// Aggregated field from all variants
    pub integrated_field: Vec<f64>,
    
    /// Overall certainty/confidence
    pub confidence: f64,
    
    /// Configuration
    #[serde(skip)]
    config: BeingConfig,
}

impl Being {
    /// Create a new Being with the given configuration
    pub fn new(config: &BeingConfig, field_config: &FieldConfig) -> Self {
        let field_dim = config.field_dim;
        let attractor_capacity = config.attractor_capacity;
        
        Self {
            master_phase: 0.0,
            field_topology: Field3D::new(field_config.resolution),
            resonance_freq: config.resonance_freq,
            attractor_state: vec![0.0; attractor_capacity * field_dim],
            integrated_field: vec![0.0; field_dim],
            confidence: 0.0,
            config: config.clone(),
        }
    }

    /// Broadcast global state to all variants
    pub fn broadcast(&self) -> Broadcast {
        Broadcast {
            global_rhythm: self.master_phase,
            field_gradient: self.field_topology.gradient(),
            attractor_state: self.attractor_state.clone(),
            resonance_freq: self.resonance_freq,
        }
    }

    /// Advance the master phase by dt
    pub fn tick(&mut self, dt: f64) {
        self.master_phase += self.resonance_freq * dt * TWO_PI;
        self.master_phase = math::wrap_angle(self.master_phase);
    }

    /// Integrate contributions from all variants
    pub fn integrate(&mut self, variants: &[Variant]) {
        let field_dim = self.config.field_dim;
        
        // Reset integrated field
        self.integrated_field = vec![0.0; field_dim];
        
        // Calculate total resonance for normalization
        let total_resonance: f64 = variants
            .iter()
            .map(|v| v.resonance_strength.max(0.01))
            .sum();
        
        // Weighted integration based on resonance strength
        for variant in variants {
            let weight = variant.resonance_strength.max(0.01) / total_resonance;
            
            // Project variant's local field to Being's field dimension
            let projected = self.project_to_field_dim(&variant.local_field);
            
            for (i, val) in projected.iter().enumerate() {
                if i < self.integrated_field.len() {
                    self.integrated_field[i] += weight * val;
                }
            }
        }
        
        // Update confidence based on coherence
        self.confidence = self.compute_coherence(variants);
        
        // Update field topology based on integrated field
        self.update_field_topology();
    }

    /// Project a vector to the Being's field dimension
    fn project_to_field_dim(&self, v: &[f64]) -> Vec<f64> {
        let field_dim = self.config.field_dim;
        let mut result = vec![0.0; field_dim];
        
        // Simple projection: repeat or truncate
        for i in 0..field_dim {
            result[i] = v[i % v.len()];
        }
        
        result
    }

    /// Compute coherence - how well variants agree
    ///
    /// Uses Kuramoto order parameter: r = |1/N * Σ exp(i*θ)|
    /// High coherence = variants resonating in phase
    /// Low coherence = variants out of phase (disagreement)
    pub fn compute_coherence(&self, variants: &[Variant]) -> f64 {
        if variants.is_empty() {
            return 0.0;
        }
        
        let n = variants.len() as f64;
        
        // Kuramoto order parameter
        let mut sum_cos = 0.0;
        let mut sum_sin = 0.0;
        
        for v in variants {
            let phase_diff = v.local_phase - self.master_phase;
            sum_cos += phase_diff.cos();
            sum_sin += phase_diff.sin();
        }
        
        let r = ((sum_cos / n).powi(2) + (sum_sin / n).powi(2)).sqrt();
        
        // Also consider mass-weighted coherence
        let total_mass: f64 = variants.iter().map(|v| v.mass).sum();
        let mut mass_weighted_coherence = 0.0;
        
        for v in variants {
            let phase_diff = v.local_phase - self.master_phase;
            mass_weighted_coherence += (v.mass / total_mass) * phase_diff.cos();
        }
        
        // Combine both measures
        0.6 * r + 0.4 * mass_weighted_coherence.abs()
    }

    /// Update field topology based on integrated field
    fn update_field_topology(&mut self) {
        // Inject integrated field values into the 3D field
        self.field_topology.inject_values(&self.integrated_field);
    }

    /// Find attractors in the current field state
    pub fn find_attractors(&self) -> Vec<Attractor> {
        let mut attractors = Vec::new();
        let attractor_capacity = self.config.attractor_capacity;
        let field_dim = self.config.field_dim;
        
        // Extract attractors from attractor_state
        for i in 0..attractor_capacity {
            let start = i * field_dim;
            let end = start + field_dim;
            
            if end <= self.attractor_state.len() {
                let state: Vec<f64> = self.attractor_state[start..end].to_vec();
                let strength = math::magnitude(&state);
                
                if strength > 0.1 {
                    attractors.push(Attractor {
                        id: i,
                        state,
                        strength,
                    });
                }
            }
        }
        
        // Sort by strength (descending)
        attractors.sort_by(|a, b| b.strength.partial_cmp(&a.strength).unwrap());
        
        attractors
    }

    /// Sculpt attractors based on learning signal
    pub fn sculpt_attractors(&mut self, actual: &Output, expected: &Output, lr: f64) {
        let field_dim = self.config.field_dim;
        
        // Find which attractor was activated
        let attractors = self.find_attractors();
        
        if attractors.is_empty() {
            return;
        }
        
        // Compare actual vs expected actions
        let error = compute_action_error(actual, expected);
        
        // Strengthen or weaken attractors based on correctness
        for (i, attractor) in attractors.iter().enumerate() {
            let start = attractor.id * field_dim;
            let end = start + field_dim;
            
            if end <= self.attractor_state.len() {
                let adjustment = if error < 0.1 {
                    // Correct - strengthen this attractor
                    lr * 0.1
                } else {
                    // Incorrect - weaken this attractor
                    -lr * 0.05 * (1.0 - 1.0 / (i as f64 + 1.0))
                };
                
                for j in start..end {
                    self.attractor_state[j] *= 1.0 + adjustment;
                }
            }
        }
    }

    /// Decode output from the current field state
    pub fn decode_output(&self, variants: &[Variant]) -> Output {
        let attractors = self.find_attractors();
        let actions = self.attractors_to_actions(&attractors, variants);
        let diagnosis = self.generate_diagnosis(variants);
        
        Output {
            diagnosis,
            root_cause: self.infer_root_cause(variants),
            actions,
            confidence: self.confidence,
            resonance_coherence: self.compute_coherence(variants),
        }
    }

    /// Convert attractors to action items
    fn attractors_to_actions(&self, attractors: &[Attractor], variants: &[Variant]) -> Vec<ActionItem> {
        let mut actions = Vec::new();
        
        // Find dominant domains from variants
        let dominant_domains = self.find_dominant_domains(variants);
        
        for (order, attractor) in attractors.iter().take(5).enumerate() {
            // Map attractor to action based on dominant domains
            if let Some(action) = self.map_attractor_to_action(attractor, &dominant_domains) {
                actions.push(ActionItem {
                    action: action.0,
                    params: action.1,
                    order: order + 1,
                    fallback: action.2,
                });
            }
        }
        
        // If no actions found, add fallback
        if actions.is_empty() && self.confidence < 0.7 {
            actions.push(ActionItem {
                action: "emergency_shell".to_string(),
                params: serde_json::json!({"message": "Low confidence recovery"}),
                order: 1,
                fallback: None,
            });
        }
        
        actions
    }

    /// Find dominant domains based on variant resonance
    fn find_dominant_domains(&self, variants: &[Variant]) -> Vec<(crate::variant::Domain, f64)> {
        use std::collections::HashMap;
        
        let mut domain_scores: HashMap<crate::variant::Domain, f64> = HashMap::new();
        
        for v in variants {
            *domain_scores.entry(v.domain).or_insert(0.0) += v.resonance_strength * v.mass;
        }
        
        let mut sorted: Vec<_> = domain_scores.into_iter().collect();
        sorted.sort_by(|a, b| b.1.partial_cmp(&a.1).unwrap());
        
        sorted
    }

    /// Map an attractor to an action
    fn map_attractor_to_action(
        &self,
        attractor: &Attractor,
        dominant_domains: &[(crate::variant::Domain, f64)],
    ) -> Option<(String, serde_json::Value, Option<String>)> {
        use crate::variant::Domain;
        
        if dominant_domains.is_empty() {
            return None;
        }
        
        let primary_domain = dominant_domains[0].0;
        let strength = attractor.strength;
        
        // Map domain + strength to action
        let (action, params, fallback) = match primary_domain {
            Domain::Kernel => {
                if strength > 0.8 {
                    ("load_module", serde_json::json!({"module": "auto"}), Some("emergency_shell".to_string()))
                } else {
                    ("log_and_continue", serde_json::json!({"severity": "warning", "message": "Kernel issue detected"}), None)
                }
            }
            Domain::Filesystem => {
                if strength > 0.8 {
                    ("fsck", serde_json::json!({"device": "auto", "auto_fix": false}), Some("emergency_shell".to_string()))
                } else {
                    ("remount_filesystem", serde_json::json!({"path": "/", "options": "rw"}), Some("fsck".to_string()))
                }
            }
            Domain::Service => {
                if strength > 0.8 {
                    ("disable_service", serde_json::json!({"service": "auto", "temporary": true}), Some("reboot".to_string()))
                } else {
                    ("restart_service", serde_json::json!({"service": "auto", "clean_state": false}), None)
                }
            }
            Domain::Boot => {
                if strength > 0.8 {
                    ("reboot", serde_json::json!({"mode": "recovery", "delay_seconds": 0}), Some("emergency_shell".to_string()))
                } else {
                    ("wait_and_retry", serde_json::json!({"condition": "boot_complete", "timeout_seconds": 30, "retry_action": "continue"}), None)
                }
            }
            Domain::Hardware => {
                ("load_module", serde_json::json!({"module": "auto"}), Some("notify_user".to_string()))
            }
            Domain::Config => {
                ("restore_config", serde_json::json!({"config_path": "auto"}), Some("emergency_shell".to_string()))
            }
            Domain::Network => {
                ("network_reset", serde_json::json!({"interface": "all"}), None)
            }
            Domain::Memory => {
                ("clear_cache", serde_json::json!({"cache_type": "all"}), Some("reboot".to_string()))
            }
        };
        
        Some((action.to_string(), params, fallback))
    }

    /// Generate diagnosis from variant states
    fn generate_diagnosis(&self, variants: &[Variant]) -> String {
        let dominant = self.find_dominant_domains(variants);
        
        if dominant.is_empty() {
            return "Unable to determine issue".to_string();
        }
        
        let primary = dominant[0].0;
        let confidence_level = if self.confidence > 0.8 {
            "High confidence"
        } else if self.confidence > 0.5 {
            "Moderate confidence"
        } else {
            "Low confidence"
        };
        
        format!(
            "{}: {} domain issue detected with resonance strength {:.2}",
            confidence_level,
            format!("{:?}", primary),
            dominant[0].1
        )
    }

    /// Infer root cause from variant states
    fn infer_root_cause(&self, variants: &[Variant]) -> Option<String> {
        let dominant = self.find_dominant_domains(variants);
        
        if dominant.len() < 2 {
            return None;
        }
        
        // If multiple domains resonate, there might be a causal chain
        let primary = dominant[0].0;
        let secondary = dominant[1].0;
        
        Some(format!(
            "Primary: {:?} issue, possibly caused by {:?} problem",
            primary, secondary
        ))
    }

    /// Reset Being to initial state
    pub fn reset(&mut self) {
        self.master_phase = 0.0;
        self.integrated_field.fill(0.0);
        self.confidence = 0.0;
        self.field_topology.clear();
    }

    /// Get the current energy of the Being's field
    pub fn energy(&self) -> f64 {
        self.field_topology.energy() + math::magnitude(&self.integrated_field)
    }
}

/// An attractor in the solution landscape
#[derive(Debug, Clone)]
pub struct Attractor {
    pub id: usize,
    pub state: Vec<f64>,
    pub strength: f64,
}

/// Compute error between actual and expected output
fn compute_action_error(actual: &Output, expected: &Output) -> f64 {
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
        }
    }
    
    1.0 - (matches as f64 / total as f64)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{VariantsConfig, CouplingConfig};

    #[test]
    fn test_being_creation() {
        let config = BeingConfig::default();
        let field_config = FieldConfig::default();
        let being = Being::new(&config, &field_config);
        
        assert_eq!(being.master_phase, 0.0);
        assert_eq!(being.resonance_freq, 1.0);
        assert_eq!(being.confidence, 0.0);
    }

    #[test]
    fn test_broadcast() {
        let config = BeingConfig::default();
        let field_config = FieldConfig::default();
        let being = Being::new(&config, &field_config);
        
        let broadcast = being.broadcast();
        assert_eq!(broadcast.global_rhythm, 0.0);
        assert_eq!(broadcast.resonance_freq, 1.0);
    }

    #[test]
    fn test_tick() {
        let config = BeingConfig::default();
        let field_config = FieldConfig::default();
        let mut being = Being::new(&config, &field_config);
        
        being.tick(0.1);
        assert!(being.master_phase > 0.0);
    }

    #[test]
    fn test_coherence_empty() {
        let config = BeingConfig::default();
        let field_config = FieldConfig::default();
        let being = Being::new(&config, &field_config);
        
        let coherence = being.compute_coherence(&[]);
        assert_eq!(coherence, 0.0);
    }
}
