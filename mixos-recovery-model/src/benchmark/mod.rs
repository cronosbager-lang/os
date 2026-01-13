//! Benchmark Suite for MRM-64
//!
//! Comprehensive benchmarking for:
//! - Accuracy & F1 Score
//! - Reasoning quality (multi-step, causal)
//! - Action correctness (sequence, params)
//! - Confidence calibration (ECE)
//! - Latency & throughput
//! - Domain-specific performance

pub mod metrics;
pub mod runner;
pub mod report;

pub use metrics::*;
pub use runner::BenchmarkRunner;
pub use report::BenchmarkReport;
