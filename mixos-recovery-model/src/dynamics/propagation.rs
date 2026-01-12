//! Field Propagation - Information Spread
//!
//! Implements field dynamics for information propagation:
//! ∂φ/∂t = D·∇²φ + Σᵢ mᵢ·δ(x - xᵢ)·cos(θᵢ)
//!
//! Information spreads through the field via diffusion,
//! with synchronized variants amplifying the field.

use crate::{math, FieldConfig, VariantState};
use crate::field::Field3D;

/// Field propagation dynamics
pub struct FieldPropagation {
    /// Diffusion coefficient
    pub diffusion: f64,
    
    /// Damping factor
    pub damping: f64,
    
    /// Field resolution
    pub resolution: usize,
}

impl FieldPropagation {
    /// Create new field propagation
    pub fn new(config: &FieldConfig) -> Self {
        Self {
            diffusion: config.diffusion,
            damping: config.damping,
            resolution: config.resolution,
        }
    }

    /// Inject variant contributions into the field
    ///
    /// Each variant acts as a source at its position,
    /// with strength proportional to mass × cos(phase)
    pub fn inject_sources(&self, field: &mut Field3D, variants: &[VariantState]) {
        let r = self.resolution as f64;
        
        for v in variants {
            // Map variant position to field coordinates
            let (fx, fy, fz) = self.position_to_field_coords(&v.position, r);
            
            // Source strength: mass × phase coherence
            let strength = v.mass * v.phase.cos();
            
            // Add Gaussian source
            field.add_gaussian_source(fx, fy, fz, strength, 2.0);
        }
    }

    /// Map variant position to field coordinates
    fn position_to_field_coords(&self, position: &[f64], field_size: f64) -> (f64, f64, f64) {
        // Use first 3 dimensions of position, or default to center
        let x = position.get(0).copied().unwrap_or(0.0);
        let y = position.get(1).copied().unwrap_or(0.0);
        let z = position.get(2).copied().unwrap_or(0.0);
        
        // Normalize to field coordinates
        // Assuming positions are roughly in [-5, 5] range
        let scale = field_size / 10.0;
        let offset = field_size / 2.0;
        
        let fx = (x * scale + offset).max(0.0).min(field_size - 1.0);
        let fy = (y * scale + offset).max(0.0).min(field_size - 1.0);
        let fz = (z * scale + offset).max(0.0).min(field_size - 1.0);
        
        (fx, fy, fz)
    }

    /// Propagate the field for one time step
    ///
    /// Applies diffusion and damping
    pub fn propagate(&self, field: &mut Field3D, dt: f64) {
        // Apply diffusion
        field.diffuse(self.diffusion, dt);
        
        // Apply damping
        field.damp(self.damping * dt);
    }

    /// Compute field gradient at variant positions
    pub fn sample_gradients(&self, field: &Field3D, variants: &[VariantState]) -> Vec<Vec<f64>> {
        let r = self.resolution as f64;
        
        variants
            .iter()
            .map(|v| {
                let (fx, fy, fz) = self.position_to_field_coords(&v.position, r);
                self.compute_gradient_at(field, fx, fy, fz)
            })
            .collect()
    }

    /// Compute gradient at a specific field position
    fn compute_gradient_at(&self, field: &Field3D, x: f64, y: f64, z: f64) -> Vec<f64> {
        let eps = 0.5;
        
        let dx = field.sample(x + eps, y, z) - field.sample(x - eps, y, z);
        let dy = field.sample(x, y + eps, z) - field.sample(x, y - eps, z);
        let dz = field.sample(x, y, z + eps) - field.sample(x, y, z - eps);
        
        vec![dx / (2.0 * eps), dy / (2.0 * eps), dz / (2.0 * eps)]
    }

    /// Compute field energy
    pub fn energy(&self, field: &Field3D) -> f64 {
        field.energy()
    }

    /// Compute field smoothness (gradient magnitude)
    pub fn smoothness(&self, field: &Field3D) -> f64 {
        field.smoothness()
    }

    /// Find field maxima (potential attractors)
    pub fn find_maxima(&self, field: &Field3D) -> Vec<FieldMaximum> {
        field
            .find_maxima()
            .into_iter()
            .map(|(x, y, z, value)| FieldMaximum {
                position: (x, y, z),
                value,
            })
            .collect()
    }

    /// Sample field value at variant position
    pub fn sample_at_variant(&self, field: &Field3D, variant: &VariantState) -> f64 {
        let r = self.resolution as f64;
        let (fx, fy, fz) = self.position_to_field_coords(&variant.position, r);
        field.sample(fx, fy, fz)
    }

    /// Compute field influence on variant
    ///
    /// Returns a force vector pointing towards higher field values
    pub fn compute_field_force(&self, field: &Field3D, variant: &VariantState) -> Vec<f64> {
        let r = self.resolution as f64;
        let (fx, fy, fz) = self.position_to_field_coords(&variant.position, r);
        
        // Gradient points towards increasing field
        let grad = self.compute_gradient_at(field, fx, fy, fz);
        
        // Scale by variant's phase coherence
        let scale = variant.phase.cos().abs();
        
        math::scale(&grad, scale)
    }

    /// Reset field to zero
    pub fn reset(&self, field: &mut Field3D) {
        field.clear();
    }

    /// Create a new field with current configuration
    pub fn create_field(&self) -> Field3D {
        Field3D::new(self.resolution)
    }
}

/// A local maximum in the field
#[derive(Debug, Clone)]
pub struct FieldMaximum {
    pub position: (usize, usize, usize),
    pub value: f64,
}

/// Result of field propagation step
#[derive(Debug, Clone)]
pub struct PropagationResult {
    /// Field energy after propagation
    pub energy: f64,
    
    /// Field smoothness after propagation
    pub smoothness: f64,
    
    /// Number of maxima found
    pub num_maxima: usize,
}

impl FieldPropagation {
    /// Perform a complete propagation step and return results
    pub fn step(
        &self,
        field: &mut Field3D,
        variants: &[VariantState],
        dt: f64,
    ) -> PropagationResult {
        // Inject sources from variants
        self.inject_sources(field, variants);
        
        // Propagate field
        self.propagate(field, dt);
        
        // Compute results
        PropagationResult {
            energy: self.energy(field),
            smoothness: self.smoothness(field),
            num_maxima: self.find_maxima(field).len(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn create_test_config() -> FieldConfig {
        FieldConfig {
            resolution: 16,
            diffusion: 0.1,
            damping: 0.05,
        }
    }

    fn create_test_variants() -> Vec<VariantState> {
        vec![
            VariantState {
                id: 0,
                phase: 0.0,
                position: vec![0.0, 0.0, 0.0],
                mass: 1.0,
                local_field: vec![],
            },
            VariantState {
                id: 1,
                phase: std::f64::consts::PI,
                position: vec![2.0, 0.0, 0.0],
                mass: 2.0,
                local_field: vec![],
            },
        ]
    }

    #[test]
    fn test_inject_sources() {
        let config = create_test_config();
        let propagation = FieldPropagation::new(&config);
        let mut field = propagation.create_field();
        let variants = create_test_variants();
        
        let initial_energy = propagation.energy(&field);
        
        propagation.inject_sources(&mut field, &variants);
        
        let final_energy = propagation.energy(&field);
        
        // Energy should increase after injection
        assert!(final_energy > initial_energy);
    }

    #[test]
    fn test_propagate() {
        let config = create_test_config();
        let propagation = FieldPropagation::new(&config);
        let mut field = propagation.create_field();
        
        // Add a point source
        field.add_gaussian_source(8.0, 8.0, 8.0, 10.0, 1.0);
        
        let initial_smoothness = propagation.smoothness(&field);
        
        // Propagate
        propagation.propagate(&mut field, 0.1);
        
        let final_smoothness = propagation.smoothness(&field);
        
        // Field should become smoother after diffusion
        // (gradient magnitude decreases)
        assert!(final_smoothness <= initial_smoothness);
    }

    #[test]
    fn test_sample_at_variant() {
        let config = create_test_config();
        let propagation = FieldPropagation::new(&config);
        let mut field = propagation.create_field();
        let variants = create_test_variants();
        
        // Add source at variant 0's position
        propagation.inject_sources(&mut field, &variants);
        
        let value = propagation.sample_at_variant(&field, &variants[0]);
        
        // Should have non-zero value at source position
        assert!(value.abs() > 0.0);
    }

    #[test]
    fn test_step() {
        let config = create_test_config();
        let propagation = FieldPropagation::new(&config);
        let mut field = propagation.create_field();
        let variants = create_test_variants();
        
        let result = propagation.step(&mut field, &variants, 0.1);
        
        assert!(result.energy > 0.0);
    }
}
