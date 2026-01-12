//! Inference Engine - ResonanceField
//!
//! The main inference system that combines Being, Variants, and Field dynamics.

use rayon::prelude::*;

use crate::{
    math, Input, Output, MrmConfig, SettleResult, VariantState,
    BeingConfig, VariantsConfig, CouplingConfig, FieldConfig, SettlingConfig,
};
use crate::being::Being;
use crate::variant::Variant;
use crate::field::Field3D;
use crate::dynamics::{KuramotoDynamics, GravityDynamics, FieldPropagation};
use crate::learning::{HebbianLearning, ResonanceTuning, AttractorSculpting};

/// The complete Resonance Field system
///
/// Combines Being (orchestrator) with Variants (specialists)
/// using Kuramoto synchronization and Gravity dynamics.
pub struct ResonanceField {
    /// The Being (orchestrator)
    pub being: Being,
    
    /// All variants (domain specialists)
    pub variants: Vec<Variant>,
    
    /// Coupling matrix between variants
    pub coupling_matrix: Vec<Vec<f64>>,
    
    /// 3D field for information propagation
    pub field: Field3D,
    
    /// Configuration
    pub config: MrmConfig,
    
    /// Kuramoto dynamics
    kuramoto: KuramotoDynamics,
    
    /// Gravity dynamics
    gravity: GravityDynamics,
    
    /// Field propagation
    propagation: FieldPropagation,
    
    /// Previous energy (for convergence check)
    prev_energy: f64,
}

impl ResonanceField {
    /// Create a new ResonanceField with default configuration
    pub fn new() -> Self {
        Self::with_config(MrmConfig::default())
    }

    /// Create a new ResonanceField with custom configuration
    pub fn with_config(config: MrmConfig) -> Self {
        let being = Being::new(&config.being, &config.field);
        let variants = Variant::create_all(&config.variants);
        
        // Initialize coupling matrix
        let hebbian = HebbianLearning::new(0.01);
        let coupling_matrix = hebbian.initialize_coupling(
            config.variants.count,
            config.variants.domains,
            config.coupling.intra_domain,
            config.coupling.inter_domain,
        );
        
        let field = Field3D::new(config.field.resolution);
        
        let kuramoto = KuramotoDynamics::new(&config.coupling);
        let gravity = GravityDynamics::new(config.field.damping);
        let propagation = FieldPropagation::new(&config.field);
        
        Self {
            being,
            variants,
            coupling_matrix,
            field,
            config,
            kuramoto,
            gravity,
            propagation,
            prev_energy: f64::MAX,
        }
    }

    /// Run inference on an input
    pub fn infer(&mut self, input: &Input) -> Output {
        // 1. Reset state for new inference
        self.reset_for_inference();
        
        // 2. Encode input
        self.encode_input(input);
        
        // 3. Being broadcasts initial state
        let broadcast = self.being.broadcast();
        
        // 4. Variants receive and process (parallel)
        self.variants.par_iter_mut().for_each(|v| {
            v.receive_and_process(&broadcast, input);
        });
        
        // 5. Field settling loop
        let settle_result = self.settle();
        
        // 6. Being integrates variant contributions
        self.being.integrate(&self.variants);
        
        // 7. Decode output
        self.being.decode_output(&self.variants)
    }

    /// Reset state for a new inference
    fn reset_for_inference(&mut self) {
        self.field.clear();
        self.prev_energy = f64::MAX;
        
        for v in &mut self.variants {
            v.local_field.fill(0.0);
            v.resonance_strength = 0.0;
        }
    }

    /// Encode input into the field
    fn encode_input(&mut self, input: &Input) {
        // Encode error message into field
        let error_hash = self.hash_string(&input.error);
        
        // Add sources based on error content
        let r = self.config.field.resolution as f64;
        let x = (error_hash % 1000) as f64 / 1000.0 * r;
        let y = ((error_hash / 1000) % 1000) as f64 / 1000.0 * r;
        let z = ((error_hash / 1000000) % 1000) as f64 / 1000.0 * r;
        
        self.field.add_gaussian_source(x, y, z, 1.0, 3.0);
        
        // Add context-based modulation
        let stage_factor = match input.context.boot_stage {
            crate::BootStage::Bootloader => 0.2,
            crate::BootStage::Kernel => 0.4,
            crate::BootStage::Init => 0.6,
            crate::BootStage::Services => 0.8,
            crate::BootStage::Ready => 1.0,
        };
        
        self.field.add_gaussian_source(r / 2.0, r / 2.0, stage_factor * r, 0.5, 2.0);
    }

    /// Simple string hash
    fn hash_string(&self, s: &str) -> u64 {
        s.bytes().fold(0u64, |acc, b| acc.wrapping_mul(31).wrapping_add(b as u64))
    }

    /// Run the field settling loop
    pub fn settle(&mut self) -> SettleResult {
        let max_steps = self.config.settling.max_steps;
        let threshold = self.config.settling.convergence_threshold;
        let dt = self.config.settling.dt;
        
        for step in 0..max_steps {
            // Perform one settling step
            self.settle_step(dt);
            
            // Check convergence
            let energy = self.total_energy();
            let delta = (self.prev_energy - energy).abs();
            
            if delta < threshold {
                return SettleResult::Converged { steps: step, energy };
            }
            
            self.prev_energy = energy;
        }
        
        SettleResult::MaxSteps { energy: self.total_energy() }
    }

    /// Perform one settling step
    fn settle_step(&mut self, dt: f64) {
        // Get snapshot of current state for parallel processing
        let snapshot: Vec<VariantState> = self.variants
            .iter()
            .map(|v| v.state())
            .collect();
        
        let master_phase = self.being.master_phase;
        let coupling_matrix = &self.coupling_matrix;
        let gravity = &self.gravity;
        
        // Parallel update of all variants
        self.variants.par_iter_mut().enumerate().for_each(|(i, v)| {
            // 1. Kuramoto phase update
            let phase_coupling: f64 = snapshot.iter().enumerate()
                .filter(|(j, _)| *j != i)
                .map(|(j, other)| {
                    let k_ij = coupling_matrix.get(i)
                        .and_then(|row| row.get(j))
                        .copied()
                        .unwrap_or(0.3);
                    k_ij * (other.phase - v.local_phase).sin()
                })
                .sum();
            
            let n = snapshot.len() as f64;
            v.local_phase += (v.natural_freq + phase_coupling / n) * dt;
            v.local_phase = math::wrap_angle(v.local_phase);
            
            // 2. Gravity position update
            let force = gravity.compute_force(i, &v.position, &snapshot);
            v.apply_gravity(&force, gravity.damping, dt);
            
            // 3. Mass update based on resonance
            let resonance = v.compute_resonance(&snapshot);
            v.update_mass(resonance, dt);
            v.resonance_strength = resonance;
        });
        
        // Update Being's master phase
        self.being.tick(dt);
        
        // Propagate field
        let variant_states: Vec<VariantState> = self.variants
            .iter()
            .map(|v| v.state())
            .collect();
        self.propagation.step(&mut self.field, &variant_states, dt);
    }

    /// Compute total system energy
    pub fn total_energy(&self) -> f64 {
        let states: Vec<VariantState> = self.variants
            .iter()
            .map(|v| v.state())
            .collect();
        
        let e_kuramoto = self.kuramoto.energy(&states, &self.coupling_matrix);
        let e_gravity = self.gravity.energy(&states);
        let e_field = self.propagation.energy(&self.field);
        
        e_kuramoto + e_gravity + e_field
    }

    /// Check if the system has converged
    pub fn is_converged(&self) -> bool {
        let energy = self.total_energy();
        (self.prev_energy - energy).abs() < self.config.settling.convergence_threshold
    }

    /// Get the current order parameter (synchronization measure)
    pub fn order_parameter(&self) -> f64 {
        let states: Vec<VariantState> = self.variants
            .iter()
            .map(|v| v.state())
            .collect();
        
        self.kuramoto.order_parameter(&states)
    }

    /// Get coherence (how well variants agree)
    pub fn coherence(&self) -> f64 {
        self.being.compute_coherence(&self.variants)
    }

    /// Learn from a training example
    pub fn learn(&mut self, input: &Input, expected: &Output, learning_rate: f64) {
        // Run inference
        let actual = self.infer(input);
        
        // Compute error
        let error = crate::learning::compute_error(&actual, expected);
        
        // Apply learning
        let hebbian = HebbianLearning::new(learning_rate);
        let resonance = ResonanceTuning::new(learning_rate * 0.1);
        let attractor = AttractorSculpting::new(learning_rate * 0.5);
        
        // 1. Hebbian coupling adjustment
        let states: Vec<VariantState> = self.variants.iter().map(|v| v.state()).collect();
        hebbian.update_coupling(&mut self.coupling_matrix, &states, error);
        
        // 2. Resonance tuning
        resonance.tune_frequencies(&mut self.variants, &actual, error);
        
        // 3. Attractor sculpting
        attractor.sculpt(&mut self.being, &actual, expected, error);
    }

    /// Reset the entire system
    pub fn reset(&mut self) {
        self.being.reset();
        self.field.clear();
        self.prev_energy = f64::MAX;
        
        for v in &mut self.variants {
            v.reset();
        }
    }

    /// Get variant by domain
    pub fn variants_by_domain(&self, domain: crate::variant::Domain) -> Vec<&Variant> {
        self.variants.iter().filter(|v| v.domain == domain).collect()
    }

    /// Get the strongest resonating variants
    pub fn strongest_variants(&self, n: usize) -> Vec<&Variant> {
        let mut sorted: Vec<_> = self.variants.iter().collect();
        sorted.sort_by(|a, b| {
            b.resonance_strength.partial_cmp(&a.resonance_strength).unwrap()
        });
        sorted.into_iter().take(n).collect()
    }

    /// Get statistics about the current state
    pub fn stats(&self) -> ResonanceStats {
        let states: Vec<VariantState> = self.variants.iter().map(|v| v.state()).collect();
        
        ResonanceStats {
            order_parameter: self.kuramoto.order_parameter(&states),
            coherence: self.being.confidence,
            total_energy: self.total_energy(),
            field_energy: self.propagation.energy(&self.field),
            mean_mass: states.iter().map(|s| s.mass).sum::<f64>() / states.len() as f64,
            mean_resonance: self.variants.iter().map(|v| v.resonance_strength).sum::<f64>() 
                / self.variants.len() as f64,
        }
    }
}

impl Default for ResonanceField {
    fn default() -> Self {
        Self::new()
    }
}

/// Statistics about the resonance field state
#[derive(Debug, Clone)]
pub struct ResonanceStats {
    pub order_parameter: f64,
    pub coherence: f64,
    pub total_energy: f64,
    pub field_energy: f64,
    pub mean_mass: f64,
    pub mean_resonance: f64,
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{Context, SystemState, BootStage};

    fn create_test_input() -> Input {
        Input {
            error: "Kernel panic - VFS: Unable to mount root fs".to_string(),
            context: Context {
                kernel_version: Some("6.1.0".to_string()),
                boot_stage: BootStage::Init,
                last_action: None,
                uptime_seconds: Some(10),
            },
            state: SystemState {
                memory_available: true,
                root_mounted: false,
                network_up: false,
                services_started: vec![],
            },
        }
    }

    #[test]
    fn test_resonance_field_creation() {
        let rf = ResonanceField::new();
        
        assert_eq!(rf.variants.len(), 64);
        assert_eq!(rf.coupling_matrix.len(), 64);
    }

    #[test]
    fn test_inference() {
        let mut rf = ResonanceField::new();
        let input = create_test_input();
        
        let output = rf.infer(&input);
        
        assert!(!output.diagnosis.is_empty());
        assert!(output.confidence >= 0.0 && output.confidence <= 1.0);
    }

    #[test]
    fn test_settle() {
        let mut rf = ResonanceField::new();
        let input = create_test_input();
        
        rf.encode_input(&input);
        
        let result = rf.settle();
        
        match result {
            SettleResult::Converged { steps, energy } => {
                assert!(steps > 0);
                assert!(energy.is_finite());
            }
            SettleResult::MaxSteps { energy } => {
                assert!(energy.is_finite());
            }
        }
    }

    #[test]
    fn test_order_parameter() {
        let rf = ResonanceField::new();
        
        let r = rf.order_parameter();
        
        // Should be between 0 and 1
        assert!(r >= 0.0 && r <= 1.0);
    }

    #[test]
    fn test_stats() {
        let rf = ResonanceField::new();
        
        let stats = rf.stats();
        
        assert!(stats.order_parameter >= 0.0);
        assert!(stats.mean_mass > 0.0);
    }

    #[test]
    fn test_learn() {
        let mut rf = ResonanceField::new();
        let input = create_test_input();
        
        let expected = Output {
            diagnosis: "Filesystem issue".to_string(),
            root_cause: Some("Missing driver".to_string()),
            actions: vec![crate::ActionItem {
                action: "load_module".to_string(),
                params: serde_json::json!({"module": "ext4"}),
                order: 1,
                fallback: None,
            }],
            confidence: 0.9,
            resonance_coherence: 0.9,
        };
        
        // Should not panic
        rf.learn(&input, &expected, 0.01);
    }
}
