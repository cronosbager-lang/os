//! Tests for Variant specialists

use mrm::variant::{Variant, Domain};
use mrm::VariantsConfig;

#[test]
fn test_variant_creation() {
    let config = VariantsConfig::default();
    let variants = Variant::create_all(&config);
    
    assert_eq!(variants.len(), config.count);
}

#[test]
fn test_variant_domains() {
    let config = VariantsConfig::default();
    let variants = Variant::create_all(&config);
    
    // Check all domains are represented
    let kernel_count = variants.iter().filter(|v| v.domain == Domain::Kernel).count();
    assert!(kernel_count > 0);
}

#[test]
fn test_variant_state() {
    let config = VariantsConfig::default();
    let variants = Variant::create_all(&config);
    
    let state = variants[0].state();
    assert_eq!(state.id, 0);
    assert!(state.mass > 0.0);
}
