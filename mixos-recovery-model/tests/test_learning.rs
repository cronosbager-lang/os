//! Tests for learning mechanisms

use mrm::learning::{HebbianLearning, compute_error};
use mrm::{Output, ActionItem};

#[test]
fn test_hebbian_coupling_init() {
    let hebbian = HebbianLearning::new(0.01);
    let coupling = hebbian.initialize_coupling(64, 8, 0.8, 0.3);
    
    assert_eq!(coupling.len(), 64);
    assert_eq!(coupling[0].len(), 64);
}

#[test]
fn test_compute_error_identical() {
    let output = Output {
        diagnosis: "Test".to_string(),
        root_cause: None,
        actions: vec![ActionItem {
            action: "test".to_string(),
            params: serde_json::json!({}),
            order: 1,
            fallback: None,
        }],
        confidence: 0.9,
        resonance_coherence: 0.9,
    };
    
    let error = compute_error(&output, &output);
    assert!(error < 0.5); // Should be low for identical outputs
}

#[test]
fn test_compute_error_different() {
    let actual = Output {
        diagnosis: "Test".to_string(),
        root_cause: None,
        actions: vec![ActionItem {
            action: "action_a".to_string(),
            params: serde_json::json!({}),
            order: 1,
            fallback: None,
        }],
        confidence: 0.9,
        resonance_coherence: 0.9,
    };
    
    let expected = Output {
        diagnosis: "Test".to_string(),
        root_cause: None,
        actions: vec![ActionItem {
            action: "action_b".to_string(),
            params: serde_json::json!({}),
            order: 1,
            fallback: None,
        }],
        confidence: 0.9,
        resonance_coherence: 0.9,
    };
    
    let error = compute_error(&actual, &expected);
    assert!(error > 0.0); // Should be non-zero for different actions
}
