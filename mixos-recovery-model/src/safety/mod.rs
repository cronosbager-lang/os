//! Safety Module
//!
//! Guardrails and safety checks for action execution.

pub mod guardrails;

pub use guardrails::{SafetyGuard, SafetyLevel, SafetyResult};
