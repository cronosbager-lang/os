//! Hebbian Learning - Coupling Adjustment
//!
//! "Neurons that fire together, wire together"
//!
//! Strengthens connections between variants that resonate together
//! on correct answers, weakens connections on incorrect answers.
//!
//! ΔKᵢⱼ ∝ cos(θᵢ - θⱼ) × correctness × mᵢ × mⱼ

use crate::VariantState;

/// Hebbian learning for coupling adjustment
pub struct HebbianLearning {
    /// Learning rate
    pub learning_rate: f64,
    
    /// Coupling decay rate (regularization)
    pub decay_rate: f64,
    
    /// Minimum coupling value
    pub min_coupling: f64,
    
    /// Maximum coupling value
    pub max_coupling: f64,
}

impl HebbianLearning {
    /// Create new Hebbian learning
    pub fn new(learning_rate: f64) -> Self {
        Self {
            learning_rate,
            decay_rate: 0.001,
            min_coupling: -1.0,
            max_coupling: 1.0,
        }
    }

    /// Update coupling matrix based on variant states and error
    ///
    /// ΔKᵢⱼ = lr × cos(θᵢ - θⱼ) × correctness × mᵢ × mⱼ
    pub fn update_coupling(
        &self,
        coupling_matrix: &mut Vec<Vec<f64>>,
        variants: &[VariantState],
        error: f64,
    ) {
        let n = variants.len();
        
        // Correctness signal: positive if correct, negative if wrong
        let correctness = if error < 0.1 {
            1.0
        } else if error < 0.5 {
            0.5 - error
        } else {
            -0.5
        };
        
        for i in 0..n {
            for j in 0..n {
                if i == j {
                    continue;
                }
                
                if i >= coupling_matrix.len() || j >= coupling_matrix[i].len() {
                    continue;
                }
                
                // Phase correlation
                let phase_corr = (variants[i].phase - variants[j].phase).cos();
                
                // Mass weighting
                let mass_weight = variants[i].mass * variants[j].mass;
                
                // Hebbian update
                let delta = self.learning_rate 
                    * phase_corr 
                    * correctness 
                    * mass_weight.sqrt();
                
                // Apply update with decay
                coupling_matrix[i][j] += delta;
                coupling_matrix[i][j] -= self.decay_rate * coupling_matrix[i][j];
                
                // Clamp to valid range
                coupling_matrix[i][j] = coupling_matrix[i][j]
                    .clamp(self.min_coupling, self.max_coupling);
            }
        }
    }

    /// Compute the Hebbian correlation between two variants
    pub fn correlation(&self, v1: &VariantState, v2: &VariantState) -> f64 {
        let phase_corr = (v1.phase - v2.phase).cos();
        let mass_weight = (v1.mass * v2.mass).sqrt();
        
        phase_corr * mass_weight
    }

    /// Compute average coupling strength
    pub fn average_coupling(&self, coupling_matrix: &[Vec<f64>]) -> f64 {
        let mut sum = 0.0;
        let mut count = 0;
        
        for row in coupling_matrix {
            for &val in row {
                sum += val.abs();
                count += 1;
            }
        }
        
        if count > 0 {
            sum / count as f64
        } else {
            0.0
        }
    }

    /// Find strongly coupled pairs
    pub fn find_strong_pairs(
        &self,
        coupling_matrix: &[Vec<f64>],
        threshold: f64,
    ) -> Vec<(usize, usize, f64)> {
        let mut pairs = Vec::new();
        
        for (i, row) in coupling_matrix.iter().enumerate() {
            for (j, &val) in row.iter().enumerate() {
                if i < j && val.abs() > threshold {
                    pairs.push((i, j, val));
                }
            }
        }
        
        // Sort by coupling strength
        pairs.sort_by(|a, b| b.2.abs().partial_cmp(&a.2.abs()).unwrap());
        
        pairs
    }

    /// Apply weight decay to all couplings
    pub fn apply_decay(&self, coupling_matrix: &mut Vec<Vec<f64>>) {
        for row in coupling_matrix.iter_mut() {
            for val in row.iter_mut() {
                *val *= 1.0 - self.decay_rate;
            }
        }
    }

    /// Initialize coupling matrix with domain-aware structure
    pub fn initialize_coupling(
        &self,
        n_variants: usize,
        n_domains: usize,
        intra_domain: f64,
        inter_domain: f64,
    ) -> Vec<Vec<f64>> {
        let variants_per_domain = n_variants / n_domains;
        let mut matrix = vec![vec![inter_domain; n_variants]; n_variants];
        
        // Set intra-domain coupling (stronger)
        for d in 0..n_domains {
            let start = d * variants_per_domain;
            let end = start + variants_per_domain;
            
            for i in start..end.min(n_variants) {
                for j in start..end.min(n_variants) {
                    if i != j {
                        matrix[i][j] = intra_domain;
                    }
                }
            }
        }
        
        // Zero diagonal
        for i in 0..n_variants {
            matrix[i][i] = 0.0;
        }
        
        matrix
    }

    /// Compute coupling energy (for monitoring)
    pub fn coupling_energy(&self, coupling_matrix: &[Vec<f64>]) -> f64 {
        coupling_matrix
            .iter()
            .flat_map(|row| row.iter())
            .map(|&v| v * v)
            .sum::<f64>()
            .sqrt()
    }
}

/// Anti-Hebbian learning (for decorrelation)
pub struct AntiHebbianLearning {
    pub learning_rate: f64,
}

impl AntiHebbianLearning {
    pub fn new(learning_rate: f64) -> Self {
        Self { learning_rate }
    }

    /// Weaken connections between correlated variants
    pub fn decorrelate(
        &self,
        coupling_matrix: &mut Vec<Vec<f64>>,
        variants: &[VariantState],
    ) {
        let n = variants.len();
        
        for i in 0..n {
            for j in 0..n {
                if i == j {
                    continue;
                }
                
                if i >= coupling_matrix.len() || j >= coupling_matrix[i].len() {
                    continue;
                }
                
                // Phase correlation
                let phase_corr = (variants[i].phase - variants[j].phase).cos();
                
                // Anti-Hebbian: weaken if correlated
                if phase_corr > 0.5 {
                    coupling_matrix[i][j] -= self.learning_rate * phase_corr;
                    coupling_matrix[i][j] = coupling_matrix[i][j].max(-1.0);
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::TWO_PI;

    fn create_test_variants(n: usize, synchronized: bool) -> Vec<VariantState> {
        (0..n)
            .map(|i| VariantState {
                id: i,
                phase: if synchronized { 0.0 } else { (i as f64) * TWO_PI / (n as f64) },
                position: vec![0.0; 8],
                mass: 1.0,
                local_field: vec![0.0; 8],
            })
            .collect()
    }

    #[test]
    fn test_hebbian_update_correct() {
        let hebbian = HebbianLearning::new(0.1);
        let mut coupling = vec![vec![0.5; 4]; 4];
        let variants = create_test_variants(4, true); // synchronized
        
        // Low error = correct
        hebbian.update_coupling(&mut coupling, &variants, 0.05);
        
        // Coupling should increase for synchronized variants
        assert!(coupling[0][1] > 0.5);
    }

    #[test]
    fn test_hebbian_update_incorrect() {
        let hebbian = HebbianLearning::new(0.1);
        let mut coupling = vec![vec![0.5; 4]; 4];
        let variants = create_test_variants(4, true);
        
        // High error = incorrect
        hebbian.update_coupling(&mut coupling, &variants, 0.9);
        
        // Coupling should decrease
        assert!(coupling[0][1] < 0.5);
    }

    #[test]
    fn test_correlation() {
        let hebbian = HebbianLearning::new(0.1);
        
        let v1 = VariantState {
            id: 0,
            phase: 0.0,
            position: vec![],
            mass: 1.0,
            local_field: vec![],
        };
        
        let v2_sync = VariantState {
            id: 1,
            phase: 0.0,
            position: vec![],
            mass: 1.0,
            local_field: vec![],
        };
        
        let v2_anti = VariantState {
            id: 1,
            phase: std::f64::consts::PI,
            position: vec![],
            mass: 1.0,
            local_field: vec![],
        };
        
        let corr_sync = hebbian.correlation(&v1, &v2_sync);
        let corr_anti = hebbian.correlation(&v1, &v2_anti);
        
        assert!(corr_sync > 0.0);
        assert!(corr_anti < 0.0);
    }

    #[test]
    fn test_initialize_coupling() {
        let hebbian = HebbianLearning::new(0.1);
        
        let coupling = hebbian.initialize_coupling(16, 4, 0.8, 0.3);
        
        assert_eq!(coupling.len(), 16);
        assert_eq!(coupling[0].len(), 16);
        
        // Intra-domain should be stronger
        assert_eq!(coupling[0][1], 0.8); // same domain
        assert_eq!(coupling[0][4], 0.3); // different domain
        
        // Diagonal should be zero
        assert_eq!(coupling[0][0], 0.0);
    }

    #[test]
    fn test_find_strong_pairs() {
        let hebbian = HebbianLearning::new(0.1);
        
        let mut coupling = vec![vec![0.1; 4]; 4];
        coupling[0][1] = 0.9;
        coupling[1][0] = 0.9;
        coupling[2][3] = 0.8;
        coupling[3][2] = 0.8;
        
        let pairs = hebbian.find_strong_pairs(&coupling, 0.5);
        
        assert_eq!(pairs.len(), 2);
        assert_eq!(pairs[0].2, 0.9);
    }
}
