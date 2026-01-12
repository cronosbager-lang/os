//! Kuramoto Dynamics - Phase Synchronization
//!
//! Implements the Kuramoto model for coupled oscillators:
//! dθᵢ/dt = ωᵢ + (1/N) Σⱼ Kᵢⱼ sin(θⱼ - θᵢ)
//!
//! This provides natural synchronization between variants that "agree".

use crate::{math, CouplingConfig, VariantState, TWO_PI};

/// Kuramoto dynamics for phase synchronization
pub struct KuramotoDynamics {
    /// Intra-domain coupling strength
    pub intra_domain_coupling: f64,
    
    /// Inter-domain coupling strength
    pub inter_domain_coupling: f64,
    
    /// Being-variant coupling strength
    pub being_coupling: f64,
}

impl KuramotoDynamics {
    /// Create new Kuramoto dynamics
    pub fn new(config: &CouplingConfig) -> Self {
        Self {
            intra_domain_coupling: config.intra_domain,
            inter_domain_coupling: config.inter_domain,
            being_coupling: config.being_variant,
        }
    }

    /// Compute phase update for a single variant
    ///
    /// Returns dθ/dt for the variant
    pub fn compute_phase_update(
        &self,
        variant_idx: usize,
        variant_phase: f64,
        variant_natural_freq: f64,
        others: &[VariantState],
        coupling_matrix: &[Vec<f64>],
        master_phase: f64,
    ) -> f64 {
        let n = others.len() as f64;
        if n == 0.0 {
            return variant_natural_freq;
        }

        // Coupling with other variants
        let mut phase_coupling = 0.0;
        for (j, other) in others.iter().enumerate() {
            if j == variant_idx {
                continue;
            }
            
            let k_ij = if variant_idx < coupling_matrix.len() && j < coupling_matrix[variant_idx].len() {
                coupling_matrix[variant_idx][j]
            } else {
                self.inter_domain_coupling
            };
            
            let phase_diff = other.phase - variant_phase;
            phase_coupling += k_ij * phase_diff.sin();
        }
        
        // Normalize by number of oscillators
        phase_coupling /= n;
        
        // Coupling with Being (master phase)
        let being_coupling = self.being_coupling * (master_phase - variant_phase).sin();
        
        // Total phase derivative
        variant_natural_freq + phase_coupling + being_coupling
    }

    /// Compute the Kuramoto order parameter
    ///
    /// r = |1/N Σ exp(i·θⱼ)|
    ///
    /// r = 1: perfect synchronization
    /// r = 0: no synchronization
    pub fn order_parameter(&self, variants: &[VariantState]) -> f64 {
        if variants.is_empty() {
            return 0.0;
        }
        
        let n = variants.len() as f64;
        
        let sum_cos: f64 = variants.iter().map(|v| v.phase.cos()).sum();
        let sum_sin: f64 = variants.iter().map(|v| v.phase.sin()).sum();
        
        ((sum_cos / n).powi(2) + (sum_sin / n).powi(2)).sqrt()
    }

    /// Compute the mean phase of all variants
    pub fn mean_phase(&self, variants: &[VariantState]) -> f64 {
        if variants.is_empty() {
            return 0.0;
        }
        
        let n = variants.len() as f64;
        
        let sum_cos: f64 = variants.iter().map(|v| v.phase.cos()).sum();
        let sum_sin: f64 = variants.iter().map(|v| v.phase.sin()).sum();
        
        (sum_sin / n).atan2(sum_cos / n)
    }

    /// Compute the Kuramoto energy
    ///
    /// E = -Σᵢⱼ Kᵢⱼ cos(θᵢ - θⱼ)
    ///
    /// Lower energy = more synchronized
    pub fn energy(&self, variants: &[VariantState], coupling_matrix: &[Vec<f64>]) -> f64 {
        let mut energy = 0.0;
        
        for (i, vi) in variants.iter().enumerate() {
            for (j, vj) in variants.iter().enumerate() {
                if i >= j {
                    continue;
                }
                
                let k_ij = if i < coupling_matrix.len() && j < coupling_matrix[i].len() {
                    coupling_matrix[i][j]
                } else {
                    self.inter_domain_coupling
                };
                
                let phase_diff = vi.phase - vj.phase;
                energy -= k_ij * phase_diff.cos();
            }
        }
        
        energy
    }

    /// Check if the system is synchronized (order parameter > threshold)
    pub fn is_synchronized(&self, variants: &[VariantState], threshold: f64) -> bool {
        self.order_parameter(variants) > threshold
    }

    /// Compute phase coherence between two groups of variants
    pub fn group_coherence(&self, group_a: &[VariantState], group_b: &[VariantState]) -> f64 {
        if group_a.is_empty() || group_b.is_empty() {
            return 0.0;
        }
        
        let mean_a = self.mean_phase(group_a);
        let mean_b = self.mean_phase(group_b);
        
        (mean_a - mean_b).cos()
    }

    /// Compute critical coupling strength for synchronization
    ///
    /// For identical oscillators: K_c = 0
    /// For distributed frequencies: K_c ≈ 2/(π·g(0)) where g is frequency distribution
    pub fn critical_coupling(&self, freq_std: f64) -> f64 {
        if freq_std < 1e-10 {
            0.0
        } else {
            // Approximate for uniform distribution
            2.0 * freq_std / std::f64::consts::PI
        }
    }
}

/// Result of a Kuramoto simulation step
#[derive(Debug, Clone)]
pub struct KuramotoStepResult {
    /// Phase updates for each variant
    pub phase_updates: Vec<f64>,
    
    /// Current order parameter
    pub order_parameter: f64,
    
    /// Current energy
    pub energy: f64,
}

#[cfg(test)]
mod tests {
    use super::*;

    fn create_test_variants(n: usize) -> Vec<VariantState> {
        (0..n)
            .map(|i| VariantState {
                id: i,
                phase: (i as f64) * TWO_PI / (n as f64),
                position: vec![0.0; 8],
                mass: 1.0,
                local_field: vec![0.0; 8],
            })
            .collect()
    }

    #[test]
    fn test_order_parameter_synchronized() {
        let dynamics = KuramotoDynamics::new(&CouplingConfig::default());
        
        // All phases equal = perfect sync
        let variants: Vec<VariantState> = (0..10)
            .map(|i| VariantState {
                id: i,
                phase: 0.0,
                position: vec![0.0; 8],
                mass: 1.0,
                local_field: vec![0.0; 8],
            })
            .collect();
        
        let r = dynamics.order_parameter(&variants);
        assert!((r - 1.0).abs() < 0.01);
    }

    #[test]
    fn test_order_parameter_desynchronized() {
        let dynamics = KuramotoDynamics::new(&CouplingConfig::default());
        
        // Phases uniformly distributed = no sync
        let variants = create_test_variants(100);
        
        let r = dynamics.order_parameter(&variants);
        assert!(r < 0.2);
    }

    #[test]
    fn test_energy_synchronized() {
        let dynamics = KuramotoDynamics::new(&CouplingConfig::default());
        
        // Synchronized variants should have lower energy
        let sync_variants: Vec<VariantState> = (0..10)
            .map(|i| VariantState {
                id: i,
                phase: 0.0,
                position: vec![0.0; 8],
                mass: 1.0,
                local_field: vec![0.0; 8],
            })
            .collect();
        
        let desync_variants = create_test_variants(10);
        
        let coupling = vec![vec![0.5; 10]; 10];
        
        let sync_energy = dynamics.energy(&sync_variants, &coupling);
        let desync_energy = dynamics.energy(&desync_variants, &coupling);
        
        assert!(sync_energy < desync_energy);
    }

    #[test]
    fn test_phase_update() {
        let dynamics = KuramotoDynamics::new(&CouplingConfig::default());
        let variants = create_test_variants(5);
        let coupling = vec![vec![0.5; 5]; 5];
        
        let update = dynamics.compute_phase_update(
            0,
            variants[0].phase,
            1.0,
            &variants,
            &coupling,
            0.0,
        );
        
        // Should have some update due to coupling
        assert!(update.abs() > 0.0);
    }
}
