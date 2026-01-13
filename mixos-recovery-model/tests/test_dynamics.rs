//! Tests for dynamics (Kuramoto, Gravity, Field)

use mrm::dynamics::{KuramotoDynamics, GravityDynamics};
use mrm::{CouplingConfig, VariantState};

#[test]
fn test_kuramoto_order_parameter() {
    let dynamics = KuramotoDynamics::new(&CouplingConfig::default());
    
    // Synchronized variants
    let variants: Vec<VariantState> = (0..10)
        .map(|i| VariantState {
            id: i,
            phase: 0.0,
            position: vec![0.0; 8],
            mass: 1.0,
            local_field: vec![],
        })
        .collect();
    
    let r = dynamics.order_parameter(&variants);
    assert!((r - 1.0).abs() < 0.01);
}

#[test]
fn test_gravity_force() {
    let dynamics = GravityDynamics::new(0.05);
    
    let variants = vec![
        VariantState {
            id: 0,
            phase: 0.0,
            position: vec![0.0, 0.0],
            mass: 1.0,
            local_field: vec![],
        },
        VariantState {
            id: 1,
            phase: 0.0,
            position: vec![1.0, 0.0],
            mass: 1.0,
            local_field: vec![],
        },
    ];
    
    let force = dynamics.compute_force(0, &variants[0].position, &variants);
    assert!(force[0] > 0.0); // Should attract towards variant 1
}

#[test]
fn test_gravity_energy() {
    let dynamics = GravityDynamics::new(0.05);
    
    let variants = vec![
        VariantState {
            id: 0,
            phase: 0.0,
            position: vec![0.0, 0.0],
            mass: 1.0,
            local_field: vec![],
        },
        VariantState {
            id: 1,
            phase: 0.0,
            position: vec![1.0, 0.0],
            mass: 1.0,
            local_field: vec![],
        },
    ];
    
    let energy = dynamics.energy(&variants);
    assert!(energy < 0.0); // Gravitational energy is negative
}
