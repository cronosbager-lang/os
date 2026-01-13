//! Tests for inference engine

use mrm::inference::ResonanceField;
use mrm::{Input, Context, SystemState, BootStage};

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
}

#[test]
fn test_inference_produces_output() {
    let mut rf = ResonanceField::new();
    let input = create_test_input();
    
    let output = rf.infer(&input);
    
    assert!(!output.diagnosis.is_empty());
    assert!(output.confidence >= 0.0 && output.confidence <= 1.0);
}

#[test]
fn test_order_parameter_range() {
    let rf = ResonanceField::new();
    let r = rf.order_parameter();
    
    assert!(r >= 0.0 && r <= 1.0);
}

#[test]
fn test_stats() {
    let rf = ResonanceField::new();
    let stats = rf.stats();
    
    assert!(stats.order_parameter >= 0.0);
    assert!(stats.mean_mass > 0.0);
}
