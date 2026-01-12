//! Inference Module
//!
//! Contains the main inference engine (ResonanceField) and output decoding.

pub mod engine;
pub mod decode;

pub use engine::ResonanceField;
pub use decode::OutputDecoder;
