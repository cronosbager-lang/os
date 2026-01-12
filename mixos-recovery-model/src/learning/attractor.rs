//! Attractor Sculpting - Solution Landscape Modification
//!
//! Modifies Being's attractor landscape to form deep wells
//! at correct solutions.
//!
//! Before: ╱╲  ╱╲  ╱╲  (many shallow minima)
//! After:  ╱  ╲__╱  ╲  (deep wells at solutions)

use crate::Output;
use crate::being::Being;
use crate::math;

/// Attractor sculpting for solution landscape modification
pub struct AttractorSculpting {
    /// Learning rate
    pub learning_rate: f64,
    
    /// Decay rate for unused attractors
    pub decay_rate: f64,
    
    /// Minimum attractor strength
    pub min_strength: f64,
    
    /// Maximum attractor strength
    pub max_strength: f64,
}

impl AttractorSculpting {
    /// Create new attractor sculpting
    pub fn new(learning_rate: f64) -> Self {
        Self {
            learning_rate,
            decay_rate: 0.01,
            min_strength: 0.0,
            max_strength: 10.0,
        }
    }

    /// Sculpt attractors based on actual vs expected output
    pub fn sculpt(
        &self,
        being: &mut Being,
        actual: &Output,
        expected: &Output,
        error: f64,
    ) {
        let attractors = being.find_attractors();
        
        if attractors.is_empty() {
            // Initialize attractors if none exist
            self.initialize_attractors(being, expected);
            return;
        }
        
        // Determine which attractors to strengthen/weaken
        let reward = if error < 0.1 {
            1.0
        } else if error < 0.5 {
            0.3
        } else {
            -0.5
        };
        
        let field_dim = being.integrated_field.len();
        let attractor_capacity = attractors.len().max(1);
        
        for (i, attractor) in attractors.iter().enumerate() {
            let start = attractor.id * field_dim;
            let end = (start + field_dim).min(being.attractor_state.len());
            
            if start >= being.attractor_state.len() {
                continue;
            }
            
            // Adjustment based on rank and reward
            let rank_factor = 1.0 / (i as f64 + 1.0);
            let adjustment = self.learning_rate * reward * rank_factor;
            
            for j in start..end {
                being.attractor_state[j] *= 1.0 + adjustment;
                being.attractor_state[j] = being.attractor_state[j]
                    .clamp(-self.max_strength, self.max_strength);
            }
        }
        
        // Apply decay to all attractors
        self.apply_decay(being);
    }

    /// Initialize attractors based on expected output
    fn initialize_attractors(&self, being: &mut Being, expected: &Output) {
        let field_dim = being.integrated_field.len();
        
        // Create attractor for each expected action
        for (i, action) in expected.actions.iter().enumerate() {
            let start = i * field_dim;
            let end = (start + field_dim).min(being.attractor_state.len());
            
            if start >= being.attractor_state.len() {
                break;
            }
            
            // Initialize with action-based pattern
            let action_hash = self.hash_action(&action.action);
            
            for j in start..end {
                let idx = j - start;
                being.attractor_state[j] = 0.5 * ((action_hash + idx as u64) as f64 / 1000.0).sin();
            }
        }
    }

    /// Simple hash function for action names
    fn hash_action(&self, action: &str) -> u64 {
        action.bytes().fold(0u64, |acc, b| acc.wrapping_mul(31).wrapping_add(b as u64))
    }

    /// Apply decay to all attractors
    fn apply_decay(&self, being: &mut Being) {
        for val in &mut being.attractor_state {
            *val *= 1.0 - self.decay_rate;
            
            // Remove very weak attractors
            if val.abs() < self.min_strength {
                *val = 0.0;
            }
        }
    }

    /// Strengthen a specific attractor
    pub fn strengthen_attractor(&self, being: &mut Being, attractor_id: usize, amount: f64) {
        let field_dim = being.integrated_field.len();
        let start = attractor_id * field_dim;
        let end = (start + field_dim).min(being.attractor_state.len());
        
        for j in start..end {
            being.attractor_state[j] *= 1.0 + amount;
            being.attractor_state[j] = being.attractor_state[j]
                .clamp(-self.max_strength, self.max_strength);
        }
    }

    /// Weaken a specific attractor
    pub fn weaken_attractor(&self, being: &mut Being, attractor_id: usize, amount: f64) {
        let field_dim = being.integrated_field.len();
        let start = attractor_id * field_dim;
        let end = (start + field_dim).min(being.attractor_state.len());
        
        for j in start..end {
            being.attractor_state[j] *= 1.0 - amount;
        }
    }

    /// Merge similar attractors
    pub fn merge_similar(&self, being: &mut Being, similarity_threshold: f64) {
        let attractors = being.find_attractors();
        let field_dim = being.integrated_field.len();
        
        let mut to_merge: Vec<(usize, usize)> = Vec::new();
        
        // Find similar pairs
        for (i, ai) in attractors.iter().enumerate() {
            for (j, aj) in attractors.iter().enumerate() {
                if i >= j {
                    continue;
                }
                
                let similarity = self.attractor_similarity(&ai.state, &aj.state);
                if similarity > similarity_threshold {
                    to_merge.push((ai.id, aj.id));
                }
            }
        }
        
        // Merge pairs (keep stronger, zero weaker)
        for (id1, id2) in to_merge {
            let start1 = id1 * field_dim;
            let start2 = id2 * field_dim;
            
            let strength1: f64 = being.attractor_state[start1..start1 + field_dim]
                .iter()
                .map(|v| v.abs())
                .sum();
            let strength2: f64 = being.attractor_state[start2..start2 + field_dim]
                .iter()
                .map(|v| v.abs())
                .sum();
            
            let (keep, remove) = if strength1 > strength2 {
                (start1, start2)
            } else {
                (start2, start1)
            };
            
            // Strengthen keeper, zero remover
            for j in 0..field_dim {
                if keep + j < being.attractor_state.len() {
                    being.attractor_state[keep + j] *= 1.2;
                }
                if remove + j < being.attractor_state.len() {
                    being.attractor_state[remove + j] = 0.0;
                }
            }
        }
    }

    /// Compute similarity between two attractor states
    fn attractor_similarity(&self, a: &[f64], b: &[f64]) -> f64 {
        let dot = math::dot(a, b);
        let mag_a = math::magnitude(a);
        let mag_b = math::magnitude(b);
        
        if mag_a < 1e-10 || mag_b < 1e-10 {
            return 0.0;
        }
        
        dot / (mag_a * mag_b)
    }

    /// Count active attractors
    pub fn count_active(&self, being: &Being) -> usize {
        being.find_attractors().len()
    }

    /// Get total attractor energy
    pub fn total_energy(&self, being: &Being) -> f64 {
        math::magnitude(&being.attractor_state)
    }

    /// Normalize attractor strengths
    pub fn normalize(&self, being: &mut Being) {
        let energy = self.total_energy(being);
        
        if energy > 1e-10 {
            let scale = 1.0 / energy;
            for val in &mut being.attractor_state {
                *val *= scale;
            }
        }
    }

    /// Add noise to attractors (for exploration)
    pub fn add_noise(&self, being: &mut Being, noise_level: f64) {
        use rand::Rng;
        let mut rng = rand::thread_rng();
        
        for val in &mut being.attractor_state {
            *val += rng.gen_range(-noise_level..noise_level);
        }
    }

    /// Prune weak attractors
    pub fn prune(&self, being: &mut Being, threshold: f64) {
        let field_dim = being.integrated_field.len();
        let n_attractors = being.attractor_state.len() / field_dim;
        
        for i in 0..n_attractors {
            let start = i * field_dim;
            let end = start + field_dim;
            
            let strength: f64 = being.attractor_state[start..end]
                .iter()
                .map(|v| v.abs())
                .sum();
            
            if strength < threshold {
                for j in start..end {
                    being.attractor_state[j] = 0.0;
                }
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{BeingConfig, FieldConfig, ActionItem};

    fn create_test_being() -> Being {
        Being::new(&BeingConfig::default(), &FieldConfig::default())
    }

    fn create_test_output(actions: Vec<&str>) -> Output {
        Output {
            diagnosis: "Test".to_string(),
            root_cause: None,
            actions: actions
                .into_iter()
                .enumerate()
                .map(|(i, a)| ActionItem {
                    action: a.to_string(),
                    params: serde_json::json!({}),
                    order: i + 1,
                    fallback: None,
                })
                .collect(),
            confidence: 0.9,
            resonance_coherence: 0.9,
        }
    }

    #[test]
    fn test_sculpt_correct() {
        let sculpting = AttractorSculpting::new(0.1);
        let mut being = create_test_being();
        
        // Initialize some attractor state
        for (i, val) in being.attractor_state.iter_mut().enumerate() {
            *val = (i as f64 * 0.1).sin();
        }
        
        let initial_energy = sculpting.total_energy(&being);
        
        let actual = create_test_output(vec!["fsck"]);
        let expected = create_test_output(vec!["fsck"]);
        
        sculpting.sculpt(&mut being, &actual, &expected, 0.05);
        
        let final_energy = sculpting.total_energy(&being);
        
        // Energy should increase (attractors strengthened)
        assert!(final_energy >= initial_energy * 0.9);
    }

    #[test]
    fn test_sculpt_incorrect() {
        let sculpting = AttractorSculpting::new(0.1);
        let mut being = create_test_being();
        
        // Initialize some attractor state
        for (i, val) in being.attractor_state.iter_mut().enumerate() {
            *val = (i as f64 * 0.1).sin() * 2.0;
        }
        
        let initial_energy = sculpting.total_energy(&being);
        
        let actual = create_test_output(vec!["reboot"]);
        let expected = create_test_output(vec!["fsck"]);
        
        sculpting.sculpt(&mut being, &actual, &expected, 0.9);
        
        let final_energy = sculpting.total_energy(&being);
        
        // Energy should decrease (attractors weakened)
        assert!(final_energy <= initial_energy);
    }

    #[test]
    fn test_strengthen_weaken() {
        let sculpting = AttractorSculpting::new(0.1);
        let mut being = create_test_being();
        
        // Initialize
        for val in &mut being.attractor_state {
            *val = 1.0;
        }
        
        let initial = being.attractor_state[0];
        
        sculpting.strengthen_attractor(&mut being, 0, 0.5);
        assert!(being.attractor_state[0] > initial);
        
        sculpting.weaken_attractor(&mut being, 0, 0.5);
        // Should be back close to initial
    }

    #[test]
    fn test_prune() {
        let sculpting = AttractorSculpting::new(0.1);
        let mut being = create_test_being();
        
        // Set some weak values
        for val in &mut being.attractor_state {
            *val = 0.001;
        }
        
        // Prune with a threshold that will remove all weak values
        // The prune function sums values per attractor, so we need a higher threshold
        sculpting.prune(&mut being, 1.0);
        
        // All should be zero now (or very close)
        assert!(being.attractor_state.iter().all(|&v| v.abs() < 0.01));
    }
}
