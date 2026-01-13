//! Tests for Being orchestrator

use mrm::being::Being;
use mrm::{BeingConfig, FieldConfig};

#[test]
fn test_being_creation() {
    let being_config = BeingConfig::default();
    let field_config = FieldConfig::default();
    let being = Being::new(&being_config, &field_config);
    
    // attractor_state length depends on field_dim, not attractor_capacity
    assert!(being.attractor_state.len() > 0);
}

#[test]
fn test_being_broadcast() {
    let being_config = BeingConfig::default();
    let field_config = FieldConfig::default();
    let being = Being::new(&being_config, &field_config);
    
    let broadcast = being.broadcast();
    
    assert!(broadcast.resonance_freq > 0.0);
}

#[test]
fn test_being_tick() {
    let being_config = BeingConfig::default();
    let field_config = FieldConfig::default();
    let mut being = Being::new(&being_config, &field_config);
    
    let initial_phase = being.master_phase;
    being.tick(0.1);
    
    assert!(being.master_phase != initial_phase);
}
