//! Safety Guardrails
//!
//! Ensures that actions are safe to execute and prevents dangerous operations.

use crate::{Output, ActionItem};
use std::collections::HashSet;

/// Safety levels for actions
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub enum SafetyLevel {
    /// Safe to execute automatically
    Low,
    /// Requires logging but can proceed
    Medium,
    /// Requires user confirmation
    High,
    /// Should not be executed automatically
    Critical,
}

/// Result of safety check
#[derive(Debug, Clone)]
pub struct SafetyResult {
    /// Whether the action is allowed
    pub allowed: bool,
    
    /// Safety level of the action
    pub level: SafetyLevel,
    
    /// Reason if not allowed
    pub reason: Option<String>,
    
    /// Whether confirmation is required
    pub requires_confirmation: bool,
}

/// Safety guard for action validation
pub struct SafetyGuard {
    /// Actions that require confirmation
    confirmation_required: HashSet<String>,
    
    /// Actions that are blocked
    blocked_actions: HashSet<String>,
    
    /// Maximum actions per inference
    max_actions: usize,
    
    /// Minimum confidence threshold
    min_confidence: f64,
    
    /// Whether to allow destructive actions
    allow_destructive: bool,
}

impl SafetyGuard {
    /// Create a new safety guard with default settings
    pub fn new() -> Self {
        let mut confirmation_required = HashSet::new();
        confirmation_required.insert("reboot".to_string());
        confirmation_required.insert("fsck".to_string());
        confirmation_required.insert("emergency_shell".to_string());
        confirmation_required.insert("unload_module".to_string());
        confirmation_required.insert("rollback_package".to_string());
        confirmation_required.insert("network_reset".to_string());
        confirmation_required.insert("repair_store".to_string());
        
        Self {
            confirmation_required,
            blocked_actions: HashSet::new(),
            max_actions: 5,
            min_confidence: 0.5,
            allow_destructive: false,
        }
    }

    /// Create a permissive safety guard (for testing)
    pub fn permissive() -> Self {
        Self {
            confirmation_required: HashSet::new(),
            blocked_actions: HashSet::new(),
            max_actions: 10,
            min_confidence: 0.0,
            allow_destructive: true,
        }
    }

    /// Create a strict safety guard
    pub fn strict() -> Self {
        let mut blocked = HashSet::new();
        blocked.insert("reboot".to_string());
        blocked.insert("fsck".to_string());
        
        let mut confirmation = HashSet::new();
        for action in &[
            "emergency_shell", "unload_module", "rollback_package",
            "network_reset", "repair_store", "disable_service",
            "restore_config", "clear_cache",
        ] {
            confirmation.insert(action.to_string());
        }
        
        Self {
            confirmation_required: confirmation,
            blocked_actions: blocked,
            max_actions: 3,
            min_confidence: 0.7,
            allow_destructive: false,
        }
    }

    /// Check if an action is safe
    pub fn check_action(&self, action: &ActionItem) -> SafetyResult {
        // Check if blocked
        if self.blocked_actions.contains(&action.action) {
            return SafetyResult {
                allowed: false,
                level: SafetyLevel::Critical,
                reason: Some(format!("Action '{}' is blocked", action.action)),
                requires_confirmation: false,
            };
        }
        
        // Check if requires confirmation
        let requires_confirmation = self.confirmation_required.contains(&action.action);
        
        // Determine safety level
        let level = self.get_action_safety_level(&action.action);
        
        // Check for destructive actions
        if !self.allow_destructive && self.is_destructive(&action.action) {
            return SafetyResult {
                allowed: false,
                level: SafetyLevel::Critical,
                reason: Some(format!("Destructive action '{}' not allowed", action.action)),
                requires_confirmation: true,
            };
        }
        
        SafetyResult {
            allowed: true,
            level,
            reason: None,
            requires_confirmation,
        }
    }

    /// Check entire output for safety
    pub fn check_output(&self, output: &Output) -> Vec<SafetyResult> {
        let mut results = Vec::new();
        
        // Check confidence threshold
        if output.confidence < self.min_confidence {
            results.push(SafetyResult {
                allowed: false,
                level: SafetyLevel::High,
                reason: Some(format!(
                    "Confidence {:.2} below threshold {:.2}",
                    output.confidence, self.min_confidence
                )),
                requires_confirmation: true,
            });
        }
        
        // Check number of actions
        if output.actions.len() > self.max_actions {
            results.push(SafetyResult {
                allowed: false,
                level: SafetyLevel::Medium,
                reason: Some(format!(
                    "Too many actions ({} > {})",
                    output.actions.len(), self.max_actions
                )),
                requires_confirmation: true,
            });
        }
        
        // Check each action
        for action in &output.actions {
            results.push(self.check_action(action));
        }
        
        results
    }

    /// Check if all actions in output are safe
    pub fn is_safe(&self, output: &Output) -> bool {
        let results = self.check_output(output);
        results.iter().all(|r| r.allowed)
    }

    /// Filter output to only safe actions
    pub fn filter_safe(&self, output: &Output) -> Output {
        let safe_actions: Vec<ActionItem> = output.actions.iter()
            .filter(|a| self.check_action(a).allowed)
            .take(self.max_actions)
            .cloned()
            .collect();
        
        Output {
            diagnosis: output.diagnosis.clone(),
            root_cause: output.root_cause.clone(),
            actions: safe_actions,
            confidence: output.confidence,
            resonance_coherence: output.resonance_coherence,
        }
    }

    /// Get safety level for an action
    fn get_action_safety_level(&self, action: &str) -> SafetyLevel {
        match action {
            "log_and_continue" | "notify_user" => SafetyLevel::Low,
            "restart_service" | "clear_cache" | "wait_and_retry" => SafetyLevel::Low,
            "load_module" | "remount_filesystem" | "restore_config" => SafetyLevel::Medium,
            "disable_service" | "network_reset" | "repair_store" => SafetyLevel::Medium,
            "fsck" | "unload_module" | "rollback_package" => SafetyLevel::High,
            "reboot" | "emergency_shell" => SafetyLevel::High,
            _ => SafetyLevel::Medium,
        }
    }

    /// Check if action is destructive
    fn is_destructive(&self, action: &str) -> bool {
        matches!(action, "reboot" | "fsck" | "unload_module" | "rollback_package")
    }

    /// Add action to confirmation required list
    pub fn require_confirmation(&mut self, action: &str) {
        self.confirmation_required.insert(action.to_string());
    }

    /// Block an action
    pub fn block_action(&mut self, action: &str) {
        self.blocked_actions.insert(action.to_string());
    }

    /// Unblock an action
    pub fn unblock_action(&mut self, action: &str) {
        self.blocked_actions.remove(action);
    }

    /// Set maximum actions
    pub fn set_max_actions(&mut self, max: usize) {
        self.max_actions = max;
    }

    /// Set minimum confidence
    pub fn set_min_confidence(&mut self, min: f64) {
        self.min_confidence = min;
    }

    /// Allow destructive actions
    pub fn allow_destructive(&mut self, allow: bool) {
        self.allow_destructive = allow;
    }
}

impl Default for SafetyGuard {
    fn default() -> Self {
        Self::new()
    }
}

/// Validate action parameters
pub fn validate_action_params(action: &ActionItem) -> Result<(), String> {
    match action.action.as_str() {
        "reboot" => {
            let mode = action.params.get("mode")
                .and_then(|v| v.as_str())
                .unwrap_or("normal");
            
            if !["normal", "recovery", "safe"].contains(&mode) {
                return Err(format!("Invalid reboot mode: {}", mode));
            }
        }
        "fsck" => {
            if action.params.get("device").is_none() {
                return Err("fsck requires 'device' parameter".to_string());
            }
        }
        "load_module" | "unload_module" => {
            if action.params.get("module").is_none() {
                return Err("Module action requires 'module' parameter".to_string());
            }
        }
        "restart_service" | "disable_service" => {
            if action.params.get("service").is_none() {
                return Err("Service action requires 'service' parameter".to_string());
            }
        }
        "remount_filesystem" => {
            if action.params.get("path").is_none() {
                return Err("remount_filesystem requires 'path' parameter".to_string());
            }
        }
        "restore_config" => {
            if action.params.get("config_path").is_none() {
                return Err("restore_config requires 'config_path' parameter".to_string());
            }
        }
        "wait_and_retry" => {
            if action.params.get("condition").is_none() {
                return Err("wait_and_retry requires 'condition' parameter".to_string());
            }
            if action.params.get("retry_action").is_none() {
                return Err("wait_and_retry requires 'retry_action' parameter".to_string());
            }
        }
        "log_and_continue" => {
            if action.params.get("message").is_none() {
                return Err("log_and_continue requires 'message' parameter".to_string());
            }
        }
        "notify_user" => {
            if action.params.get("title").is_none() || action.params.get("message").is_none() {
                return Err("notify_user requires 'title' and 'message' parameters".to_string());
            }
        }
        _ => {}
    }
    
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn create_test_action(name: &str) -> ActionItem {
        ActionItem {
            action: name.to_string(),
            params: serde_json::json!({}),
            order: 1,
            fallback: None,
        }
    }

    fn create_test_output(actions: Vec<&str>, confidence: f64) -> Output {
        Output {
            diagnosis: "Test".to_string(),
            root_cause: None,
            actions: actions.into_iter().enumerate().map(|(i, a)| ActionItem {
                action: a.to_string(),
                params: serde_json::json!({}),
                order: i + 1,
                fallback: None,
            }).collect(),
            confidence,
            resonance_coherence: 0.9,
        }
    }

    #[test]
    fn test_check_safe_action() {
        let guard = SafetyGuard::new();
        
        let action = create_test_action("log_and_continue");
        let result = guard.check_action(&action);
        
        assert!(result.allowed);
        assert_eq!(result.level, SafetyLevel::Low);
        assert!(!result.requires_confirmation);
    }

    #[test]
    fn test_check_confirmation_required() {
        let mut guard = SafetyGuard::new();
        // Enable destructive actions for this test
        guard.allow_destructive(true);
        
        let action = create_test_action("reboot");
        let result = guard.check_action(&action);
        
        assert!(result.allowed);
        assert!(result.requires_confirmation);
    }

    #[test]
    fn test_blocked_action() {
        let mut guard = SafetyGuard::new();
        guard.block_action("dangerous_action");
        
        let action = create_test_action("dangerous_action");
        let result = guard.check_action(&action);
        
        assert!(!result.allowed);
        assert_eq!(result.level, SafetyLevel::Critical);
    }

    #[test]
    fn test_low_confidence() {
        let guard = SafetyGuard::new();
        
        let output = create_test_output(vec!["log_and_continue"], 0.3);
        let results = guard.check_output(&output);
        
        // First result should be about confidence
        assert!(!results[0].allowed);
    }

    #[test]
    fn test_too_many_actions() {
        let guard = SafetyGuard::new();
        
        let output = create_test_output(
            vec!["a", "b", "c", "d", "e", "f", "g"],
            0.9
        );
        let results = guard.check_output(&output);
        
        // Should have a result about too many actions
        assert!(results.iter().any(|r| r.reason.as_ref()
            .map(|s| s.contains("Too many"))
            .unwrap_or(false)));
    }

    #[test]
    fn test_filter_safe() {
        let mut guard = SafetyGuard::new();
        guard.block_action("dangerous");
        
        let output = create_test_output(
            vec!["log_and_continue", "dangerous", "notify_user"],
            0.9
        );
        
        let filtered = guard.filter_safe(&output);
        
        assert_eq!(filtered.actions.len(), 2);
        assert!(filtered.actions.iter().all(|a| a.action != "dangerous"));
    }

    #[test]
    fn test_validate_params() {
        let mut action = create_test_action("fsck");
        
        // Should fail without device
        assert!(validate_action_params(&action).is_err());
        
        // Should pass with device
        action.params = serde_json::json!({"device": "/dev/sda1"});
        assert!(validate_action_params(&action).is_ok());
    }

    #[test]
    fn test_strict_guard() {
        let guard = SafetyGuard::strict();
        
        let action = create_test_action("reboot");
        let result = guard.check_action(&action);
        
        // Reboot should be blocked in strict mode
        assert!(!result.allowed);
    }

    #[test]
    fn test_permissive_guard() {
        let guard = SafetyGuard::permissive();
        
        let action = create_test_action("reboot");
        let result = guard.check_action(&action);
        
        // Everything allowed in permissive mode
        assert!(result.allowed);
        assert!(!result.requires_confirmation);
    }
}
