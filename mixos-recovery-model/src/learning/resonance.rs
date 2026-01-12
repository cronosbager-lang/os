//! Resonance Tuning - Frequency Adjustment
//!
//! Adjusts natural frequencies (ωᵢ) of variants so they learn to
//! resonate at compatible frequencies for similar problem types.
//!
//! Before: ω₁=1.0, ω₂=2.3, ω₃=0.8 (dissonant)
//! After:  ω₁=1.5, ω₂=1.5, ω₃=1.5 (harmonic)

use crate::Output;
use crate::variant::Variant;

/// Resonance tuning for frequency adjustment
pub struct ResonanceTuning {
    /// Learning rate for frequency adjustment
    pub learning_rate: f64,
    
    /// Minimum frequency
    pub min_freq: f64,
    
    /// Maximum frequency
    pub max_freq: f64,
    
    /// Momentum for smooth updates
    pub momentum: f64,
    
    /// Previous frequency deltas (for momentum)
    prev_deltas: Vec<f64>,
}

impl ResonanceTuning {
    /// Create new resonance tuning
    pub fn new(learning_rate: f64) -> Self {
        Self {
            learning_rate,
            min_freq: 0.1,
            max_freq: 5.0,
            momentum: 0.9,
            prev_deltas: Vec::new(),
        }
    }

    /// Tune frequencies based on output and error
    pub fn tune_frequencies(
        &self,
        variants: &mut [Variant],
        output: &Output,
        error: f64,
    ) {
        // Reward signal: positive if correct, negative if wrong
        let reward = if error < 0.1 {
            1.0
        } else if error < 0.5 {
            0.5 - error
        } else {
            -0.3
        };
        
        for variant in variants.iter_mut() {
            // Contribution based on resonance strength
            let contribution = variant.resonance_strength;
            
            // Frequency adjustment
            let delta = self.learning_rate * reward * contribution;
            
            // Apply adjustment
            variant.natural_freq += delta;
            
            // Clamp to valid range
            variant.natural_freq = variant.natural_freq.clamp(self.min_freq, self.max_freq);
        }
    }

    /// Tune frequencies to harmonize with a target frequency
    pub fn harmonize(&self, variants: &mut [Variant], target_freq: f64) {
        for variant in variants.iter_mut() {
            let diff = target_freq - variant.natural_freq;
            let delta = self.learning_rate * diff * variant.resonance_strength;
            
            variant.natural_freq += delta;
            variant.natural_freq = variant.natural_freq.clamp(self.min_freq, self.max_freq);
        }
    }

    /// Compute frequency spread (standard deviation)
    pub fn frequency_spread(&self, variants: &[Variant]) -> f64 {
        if variants.is_empty() {
            return 0.0;
        }
        
        let mean: f64 = variants.iter().map(|v| v.natural_freq).sum::<f64>() / variants.len() as f64;
        
        let variance: f64 = variants
            .iter()
            .map(|v| (v.natural_freq - mean).powi(2))
            .sum::<f64>() / variants.len() as f64;
        
        variance.sqrt()
    }

    /// Compute mean frequency
    pub fn mean_frequency(&self, variants: &[Variant]) -> f64 {
        if variants.is_empty() {
            return 0.0;
        }
        
        variants.iter().map(|v| v.natural_freq).sum::<f64>() / variants.len() as f64
    }

    /// Find harmonic relationships between variants
    pub fn find_harmonics(&self, variants: &[Variant], tolerance: f64) -> Vec<(usize, usize, f64)> {
        let mut harmonics = Vec::new();
        
        for (i, vi) in variants.iter().enumerate() {
            for (j, vj) in variants.iter().enumerate() {
                if i >= j {
                    continue;
                }
                
                // Check for harmonic ratios (1:1, 1:2, 2:3, etc.)
                let ratio = vi.natural_freq / vj.natural_freq;
                
                let harmonic_ratios = [1.0, 0.5, 2.0, 1.5, 0.667, 0.75, 1.333];
                
                for &hr in &harmonic_ratios {
                    if (ratio - hr).abs() < tolerance {
                        harmonics.push((i, j, hr));
                        break;
                    }
                }
            }
        }
        
        harmonics
    }

    /// Encourage harmonic relationships
    pub fn encourage_harmonics(&self, variants: &mut [Variant]) {
        let n = variants.len();
        
        for i in 0..n {
            for j in (i + 1)..n {
                let ratio = variants[i].natural_freq / variants[j].natural_freq;
                
                // Find nearest harmonic ratio
                let target_ratio = self.nearest_harmonic(ratio);
                
                // Adjust frequencies towards harmonic relationship
                let adjustment = self.learning_rate * 0.1 * (target_ratio - ratio);
                
                // Apply adjustment (weighted by resonance)
                let weight_i = variants[i].resonance_strength;
                let weight_j = variants[j].resonance_strength;
                let total_weight = weight_i + weight_j + 0.01;
                
                variants[i].natural_freq *= 1.0 + adjustment * (weight_j / total_weight);
                variants[j].natural_freq *= 1.0 - adjustment * (weight_i / total_weight);
                
                // Clamp
                variants[i].natural_freq = variants[i].natural_freq.clamp(self.min_freq, self.max_freq);
                variants[j].natural_freq = variants[j].natural_freq.clamp(self.min_freq, self.max_freq);
            }
        }
    }

    /// Find nearest harmonic ratio
    fn nearest_harmonic(&self, ratio: f64) -> f64 {
        let harmonic_ratios = [1.0, 0.5, 2.0, 1.5, 0.667, 0.75, 1.333, 0.333, 3.0];
        
        harmonic_ratios
            .iter()
            .min_by(|&&a, &&b| {
                (a - ratio).abs().partial_cmp(&(b - ratio).abs()).unwrap()
            })
            .copied()
            .unwrap_or(1.0)
    }

    /// Compute resonance quality (how well frequencies align)
    pub fn resonance_quality(&self, variants: &[Variant]) -> f64 {
        if variants.len() < 2 {
            return 1.0;
        }
        
        let mut quality = 0.0;
        let mut count = 0;
        
        for (i, vi) in variants.iter().enumerate() {
            for (j, vj) in variants.iter().enumerate() {
                if i >= j {
                    continue;
                }
                
                let ratio = vi.natural_freq / vj.natural_freq;
                let nearest = self.nearest_harmonic(ratio);
                let deviation = (ratio - nearest).abs();
                
                // Quality is higher when closer to harmonic
                quality += 1.0 / (1.0 + deviation * 10.0);
                count += 1;
            }
        }
        
        if count > 0 {
            quality / count as f64
        } else {
            1.0
        }
    }

    /// Group variants by similar frequency
    pub fn group_by_frequency(&self, variants: &[Variant], tolerance: f64) -> Vec<Vec<usize>> {
        let mut groups: Vec<Vec<usize>> = Vec::new();
        let mut assigned = vec![false; variants.len()];
        
        for (i, vi) in variants.iter().enumerate() {
            if assigned[i] {
                continue;
            }
            
            let mut group = vec![i];
            assigned[i] = true;
            
            for (j, vj) in variants.iter().enumerate() {
                if assigned[j] {
                    continue;
                }
                
                if (vi.natural_freq - vj.natural_freq).abs() < tolerance {
                    group.push(j);
                    assigned[j] = true;
                }
            }
            
            groups.push(group);
        }
        
        groups
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::variant::{Domain, IterState};

    fn create_test_variants() -> Vec<Variant> {
        vec![
            Variant::new(0, Domain::Kernel, IterState::A, 8),
            Variant::new(1, Domain::Filesystem, IterState::A, 8),
            Variant::new(2, Domain::Service, IterState::A, 8),
        ]
    }

    #[test]
    fn test_tune_frequencies_correct() {
        let tuning = ResonanceTuning::new(0.1);
        let mut variants = create_test_variants();
        
        // Set resonance strengths
        for v in &mut variants {
            v.resonance_strength = 0.8;
        }
        
        let initial_freqs: Vec<f64> = variants.iter().map(|v| v.natural_freq).collect();
        
        let output = Output {
            diagnosis: String::new(),
            root_cause: None,
            actions: vec![],
            confidence: 0.9,
            resonance_coherence: 0.9,
        };
        
        tuning.tune_frequencies(&mut variants, &output, 0.05);
        
        // Frequencies should increase (reward for correct)
        for (i, v) in variants.iter().enumerate() {
            assert!(v.natural_freq >= initial_freqs[i]);
        }
    }

    #[test]
    fn test_frequency_spread() {
        let tuning = ResonanceTuning::new(0.1);
        let mut variants = create_test_variants();
        
        // Set different frequencies
        variants[0].natural_freq = 1.0;
        variants[1].natural_freq = 2.0;
        variants[2].natural_freq = 3.0;
        
        let spread = tuning.frequency_spread(&variants);
        
        assert!(spread > 0.0);
    }

    #[test]
    fn test_harmonize() {
        let tuning = ResonanceTuning::new(0.5);
        let mut variants = create_test_variants();
        
        // Set different frequencies
        variants[0].natural_freq = 0.5;
        variants[1].natural_freq = 1.5;
        variants[2].natural_freq = 2.5;
        
        // Set resonance strengths
        for v in &mut variants {
            v.resonance_strength = 1.0;
        }
        
        let target = 1.5;
        tuning.harmonize(&mut variants, target);
        
        // All frequencies should move towards target
        for v in &variants {
            let dist_to_target = (v.natural_freq - target).abs();
            assert!(dist_to_target < 1.0);
        }
    }

    #[test]
    fn test_nearest_harmonic() {
        let tuning = ResonanceTuning::new(0.1);
        
        assert!((tuning.nearest_harmonic(0.98) - 1.0).abs() < 0.01);
        assert!((tuning.nearest_harmonic(0.52) - 0.5).abs() < 0.01);
        assert!((tuning.nearest_harmonic(1.95) - 2.0).abs() < 0.01);
    }

    #[test]
    fn test_resonance_quality() {
        let tuning = ResonanceTuning::new(0.1);
        let mut variants = create_test_variants();
        
        // Set harmonic frequencies
        variants[0].natural_freq = 1.0;
        variants[1].natural_freq = 2.0;
        variants[2].natural_freq = 1.5;
        
        let quality = tuning.resonance_quality(&variants);
        
        assert!(quality > 0.5);
    }
}
