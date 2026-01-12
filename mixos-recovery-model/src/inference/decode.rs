//! Output Decoding
//!
//! Converts field state and variant resonance into actionable output.

use crate::{Output, ActionItem};
use crate::being::{Being, Attractor};
use crate::variant::{Variant, Domain};

/// Output decoder for converting field state to actions
pub struct OutputDecoder {
    /// Minimum confidence threshold
    pub min_confidence: f64,
    
    /// Maximum actions to output
    pub max_actions: usize,
    
    /// Fallback action when confidence is low
    pub fallback_action: String,
}

impl OutputDecoder {
    /// Create a new output decoder
    pub fn new() -> Self {
        Self {
            min_confidence: 0.7,
            max_actions: 5,
            fallback_action: "emergency_shell".to_string(),
        }
    }

    /// Create with custom settings
    pub fn with_settings(min_confidence: f64, max_actions: usize, fallback: &str) -> Self {
        Self {
            min_confidence,
            max_actions,
            fallback_action: fallback.to_string(),
        }
    }

    /// Decode output from Being and Variants
    pub fn decode(&self, being: &Being, variants: &[Variant]) -> Output {
        let attractors = being.find_attractors();
        let dominant_domains = self.find_dominant_domains(variants);
        let confidence = being.confidence;
        
        // Generate actions based on attractors and domains
        let actions = if confidence >= self.min_confidence {
            self.generate_actions(&attractors, &dominant_domains)
        } else {
            // Low confidence - use fallback
            vec![ActionItem {
                action: self.fallback_action.clone(),
                params: serde_json::json!({
                    "message": "Low confidence recovery - manual intervention recommended"
                }),
                order: 1,
                fallback: None,
            }]
        };
        
        // Generate diagnosis
        let diagnosis = self.generate_diagnosis(&dominant_domains, confidence);
        
        // Infer root cause
        let root_cause = self.infer_root_cause(&dominant_domains, variants);
        
        // Compute resonance coherence
        let resonance_coherence = self.compute_resonance_coherence(variants);
        
        Output {
            diagnosis,
            root_cause,
            actions,
            confidence,
            resonance_coherence,
        }
    }

    /// Find dominant domains based on variant resonance
    fn find_dominant_domains(&self, variants: &[Variant]) -> Vec<(Domain, f64)> {
        use std::collections::HashMap;
        
        let mut domain_scores: HashMap<Domain, f64> = HashMap::new();
        
        for v in variants {
            let score = v.resonance_strength * v.mass;
            *domain_scores.entry(v.domain).or_insert(0.0) += score;
        }
        
        let mut sorted: Vec<_> = domain_scores.into_iter().collect();
        sorted.sort_by(|a, b| b.1.partial_cmp(&a.1).unwrap());
        
        sorted
    }

    /// Generate actions from attractors and domains
    fn generate_actions(
        &self,
        attractors: &[Attractor],
        dominant_domains: &[(Domain, f64)],
    ) -> Vec<ActionItem> {
        let mut actions = Vec::new();
        
        if dominant_domains.is_empty() {
            return actions;
        }
        
        // Map each dominant domain to an action
        for (order, (domain, strength)) in dominant_domains.iter().take(self.max_actions).enumerate() {
            if let Some(action) = self.domain_to_action(*domain, *strength) {
                actions.push(ActionItem {
                    action: action.0,
                    params: action.1,
                    order: order + 1,
                    fallback: action.2,
                });
            }
        }
        
        actions
    }

    /// Map a domain to an action
    fn domain_to_action(
        &self,
        domain: Domain,
        strength: f64,
    ) -> Option<(String, serde_json::Value, Option<String>)> {
        let high_strength = strength > 1.0;
        
        let (action, params, fallback) = match domain {
            Domain::Kernel => {
                if high_strength {
                    (
                        "load_module",
                        serde_json::json!({"module": "auto", "params": ""}),
                        Some("emergency_shell".to_string()),
                    )
                } else {
                    (
                        "log_and_continue",
                        serde_json::json!({"severity": "warning", "message": "Kernel issue detected"}),
                        None,
                    )
                }
            }
            Domain::Filesystem => {
                if high_strength {
                    (
                        "fsck",
                        serde_json::json!({"device": "/dev/sda1", "auto_fix": false}),
                        Some("emergency_shell".to_string()),
                    )
                } else {
                    (
                        "remount_filesystem",
                        serde_json::json!({"path": "/", "options": "rw"}),
                        Some("fsck".to_string()),
                    )
                }
            }
            Domain::Service => {
                if high_strength {
                    (
                        "disable_service",
                        serde_json::json!({"service": "auto", "temporary": true}),
                        Some("reboot".to_string()),
                    )
                } else {
                    (
                        "restart_service",
                        serde_json::json!({"service": "auto", "clean_state": false}),
                        None,
                    )
                }
            }
            Domain::Boot => {
                if high_strength {
                    (
                        "reboot",
                        serde_json::json!({"mode": "recovery", "delay_seconds": 0}),
                        Some("emergency_shell".to_string()),
                    )
                } else {
                    (
                        "wait_and_retry",
                        serde_json::json!({
                            "condition": "boot_complete",
                            "timeout_seconds": 30,
                            "retry_action": "continue"
                        }),
                        None,
                    )
                }
            }
            Domain::Hardware => {
                (
                    "load_module",
                    serde_json::json!({"module": "auto"}),
                    Some("notify_user".to_string()),
                )
            }
            Domain::Config => {
                (
                    "restore_config",
                    serde_json::json!({"config_path": "/etc/mixos/config.toml"}),
                    Some("emergency_shell".to_string()),
                )
            }
            Domain::Network => {
                (
                    "network_reset",
                    serde_json::json!({"interface": "all"}),
                    None,
                )
            }
            Domain::Memory => {
                (
                    "clear_cache",
                    serde_json::json!({"cache_type": "all"}),
                    Some("reboot".to_string()),
                )
            }
        };
        
        Some((action.to_string(), params, fallback))
    }

    /// Generate diagnosis text
    fn generate_diagnosis(&self, dominant_domains: &[(Domain, f64)], confidence: f64) -> String {
        if dominant_domains.is_empty() {
            return "Unable to determine issue - insufficient resonance".to_string();
        }
        
        let primary = dominant_domains[0].0;
        let strength = dominant_domains[0].1;
        
        let confidence_level = if confidence > 0.9 {
            "Very high confidence"
        } else if confidence > 0.7 {
            "High confidence"
        } else if confidence > 0.5 {
            "Moderate confidence"
        } else {
            "Low confidence"
        };
        
        let domain_desc = match primary {
            Domain::Kernel => "kernel/driver",
            Domain::Filesystem => "filesystem/mount",
            Domain::Service => "service/daemon",
            Domain::Boot => "boot sequence",
            Domain::Hardware => "hardware/device",
            Domain::Config => "configuration",
            Domain::Network => "network",
            Domain::Memory => "memory/allocation",
        };
        
        format!(
            "{}: {} issue detected (resonance strength: {:.2})",
            confidence_level, domain_desc, strength
        )
    }

    /// Infer root cause from domain relationships
    fn infer_root_cause(&self, dominant_domains: &[(Domain, f64)], variants: &[Variant]) -> Option<String> {
        if dominant_domains.len() < 2 {
            return None;
        }
        
        let primary = dominant_domains[0].0;
        let secondary = dominant_domains[1].0;
        
        // Look for causal relationships
        let cause = match (primary, secondary) {
            (Domain::Filesystem, Domain::Kernel) => {
                "Filesystem issue likely caused by missing kernel module"
            }
            (Domain::Service, Domain::Config) => {
                "Service failure likely caused by configuration error"
            }
            (Domain::Boot, Domain::Filesystem) => {
                "Boot failure likely caused by filesystem issue"
            }
            (Domain::Network, Domain::Service) => {
                "Network issue likely caused by service dependency"
            }
            (Domain::Memory, Domain::Service) => {
                "Memory issue likely caused by service resource exhaustion"
            }
            _ => {
                return Some(format!(
                    "Primary: {:?} issue, secondary: {:?} involvement",
                    primary, secondary
                ));
            }
        };
        
        Some(cause.to_string())
    }

    /// Compute resonance coherence across all variants
    fn compute_resonance_coherence(&self, variants: &[Variant]) -> f64 {
        if variants.is_empty() {
            return 0.0;
        }
        
        let n = variants.len() as f64;
        
        // Compute phase coherence (Kuramoto order parameter)
        let sum_cos: f64 = variants.iter().map(|v| v.local_phase.cos()).sum();
        let sum_sin: f64 = variants.iter().map(|v| v.local_phase.sin()).sum();
        
        ((sum_cos / n).powi(2) + (sum_sin / n).powi(2)).sqrt()
    }
}

impl Default for OutputDecoder {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{BeingConfig, FieldConfig};
    use crate::variant::IterState;

    fn create_test_being() -> Being {
        Being::new(&BeingConfig::default(), &FieldConfig::default())
    }

    fn create_test_variants() -> Vec<Variant> {
        let mut variants = vec![
            Variant::new(0, Domain::Kernel, IterState::A, 64),
            Variant::new(1, Domain::Filesystem, IterState::A, 64),
            Variant::new(2, Domain::Service, IterState::A, 64),
        ];
        
        // Set resonance strengths
        variants[0].resonance_strength = 0.3;
        variants[0].mass = 1.0;
        variants[1].resonance_strength = 0.9;
        variants[1].mass = 2.0;
        variants[2].resonance_strength = 0.5;
        variants[2].mass = 1.5;
        
        variants
    }

    #[test]
    fn test_decode() {
        let decoder = OutputDecoder::new();
        let mut being = create_test_being();
        let variants = create_test_variants();
        
        being.confidence = 0.8;
        
        let output = decoder.decode(&being, &variants);
        
        assert!(!output.diagnosis.is_empty());
        assert!(output.confidence > 0.0);
    }

    #[test]
    fn test_low_confidence_fallback() {
        let decoder = OutputDecoder::new();
        let mut being = create_test_being();
        let variants = create_test_variants();
        
        being.confidence = 0.3; // Low confidence
        
        let output = decoder.decode(&being, &variants);
        
        assert_eq!(output.actions.len(), 1);
        assert_eq!(output.actions[0].action, "emergency_shell");
    }

    #[test]
    fn test_find_dominant_domains() {
        let decoder = OutputDecoder::new();
        let variants = create_test_variants();
        
        let dominant = decoder.find_dominant_domains(&variants);
        
        // Filesystem should be dominant (highest resonance * mass)
        assert_eq!(dominant[0].0, Domain::Filesystem);
    }

    #[test]
    fn test_resonance_coherence() {
        let decoder = OutputDecoder::new();
        let mut variants = create_test_variants();
        
        // Set all phases to same value (synchronized)
        for v in &mut variants {
            v.local_phase = 0.0;
        }
        
        let coherence = decoder.compute_resonance_coherence(&variants);
        
        // Should be close to 1.0 (perfect sync)
        assert!((coherence - 1.0).abs() < 0.01);
    }
}
