//! Dynamics Module
//!
//! Contains the three coupled dynamics:
//! - Kuramoto: Phase synchronization
//! - Gravity: Attention/clustering
//! - Propagation: Field information spread

pub mod kuramoto;
pub mod gravity;
pub mod propagation;

pub use kuramoto::KuramotoDynamics;
pub use gravity::GravityDynamics;
pub use propagation::FieldPropagation;

use crate::{VariantState, FieldConfig, CouplingConfig};

/// Combined dynamics for the field resonance system
pub struct CombinedDynamics {
    pub kuramoto: KuramotoDynamics,
    pub gravity: GravityDynamics,
    pub propagation: FieldPropagation,
}

impl CombinedDynamics {
    /// Create new combined dynamics
    pub fn new(coupling_config: &CouplingConfig, field_config: &FieldConfig) -> Self {
        Self {
            kuramoto: KuramotoDynamics::new(coupling_config),
            gravity: GravityDynamics::new(field_config.damping),
            propagation: FieldPropagation::new(field_config),
        }
    }

    /// Compute total energy of the system
    pub fn total_energy(
        &self,
        variants: &[VariantState],
        coupling_matrix: &[Vec<f64>],
    ) -> f64 {
        let e_kuramoto = self.kuramoto.energy(variants, coupling_matrix);
        let e_gravity = self.gravity.energy(variants);
        
        e_kuramoto + e_gravity
    }
}
