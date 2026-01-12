//! Variant - Domain Specialist
//!
//! Each Variant represents a specialized perspective on OS recovery.
//! Variants have unique representations but synchronize through resonance.

use crate::{math, Broadcast, Input, VariantState, VariantsConfig, TWO_PI};
use rand::Rng;
use rand_distr::{Distribution, Normal};
use serde::{Deserialize, Serialize};
use std::f64::consts::PI;

/// Domain types for Variants
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum Domain {
    Kernel,
    Filesystem,
    Service,
    Boot,
    Hardware,
    Config,
    Network,
    Memory,
}

impl Domain {
    /// Get all domains
    pub fn all() -> &'static [Domain] {
        &[
            Domain::Kernel,
            Domain::Filesystem,
            Domain::Service,
            Domain::Boot,
            Domain::Hardware,
            Domain::Config,
            Domain::Network,
            Domain::Memory,
        ]
    }

    /// Get domain index (0-7)
    pub fn index(&self) -> usize {
        match self {
            Domain::Kernel => 0,
            Domain::Filesystem => 1,
            Domain::Service => 2,
            Domain::Boot => 3,
            Domain::Hardware => 4,
            Domain::Config => 5,
            Domain::Network => 6,
            Domain::Memory => 7,
        }
    }

    /// Get domain from index
    pub fn from_index(idx: usize) -> Option<Domain> {
        match idx {
            0 => Some(Domain::Kernel),
            1 => Some(Domain::Filesystem),
            2 => Some(Domain::Service),
            3 => Some(Domain::Boot),
            4 => Some(Domain::Hardware),
            5 => Some(Domain::Config),
            6 => Some(Domain::Network),
            7 => Some(Domain::Memory),
            _ => None,
        }
    }

    /// Get keywords associated with this domain
    pub fn keywords(&self) -> &'static [&'static str] {
        match self {
            Domain::Kernel => &[
                "kernel", "panic", "oops", "module", "driver", "irq", "dma",
                "syscall", "segfault", "null pointer", "stack", "heap",
            ],
            Domain::Filesystem => &[
                "mount", "umount", "fsck", "ext4", "btrfs", "xfs", "vfs",
                "inode", "superblock", "journal", "corruption", "readonly",
            ],
            Domain::Service => &[
                "service", "systemd", "init", "daemon", "socket", "unit",
                "dependency", "timeout", "failed", "restart", "enable",
            ],
            Domain::Boot => &[
                "boot", "grub", "initramfs", "initrd", "cmdline", "root=",
                "init=", "stage", "early", "late", "target",
            ],
            Domain::Hardware => &[
                "hardware", "device", "pci", "usb", "acpi", "firmware",
                "bios", "uefi", "driver", "probe", "detect",
            ],
            Domain::Config => &[
                "config", "configuration", "syntax", "parse", "invalid",
                "missing", "required", "toml", "yaml", "json",
            ],
            Domain::Network => &[
                "network", "interface", "eth", "wlan", "ip", "dhcp",
                "dns", "route", "socket", "connection", "timeout",
            ],
            Domain::Memory => &[
                "memory", "oom", "swap", "malloc", "free", "leak",
                "allocation", "mmap", "page", "cache", "buffer",
            ],
        }
    }

    /// Get base natural frequency for this domain
    pub fn base_frequency(&self) -> f64 {
        match self {
            Domain::Kernel => 1.0,
            Domain::Filesystem => 1.1,
            Domain::Service => 0.9,
            Domain::Boot => 1.2,
            Domain::Hardware => 0.8,
            Domain::Config => 1.0,
            Domain::Network => 0.95,
            Domain::Memory => 1.05,
        }
    }
}

/// Iteration states (coarse to fine analysis)
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum IterState {
    A, B, C,  // Coarse (0° - 30°)
    D, E, F,  // Medium (40° - 70°)
    G, H, I,  // Fine (80° - 110°)
    J, K, L,  // Deep (120° - 150°)
}

impl IterState {
    /// Get all iteration states
    pub fn all() -> &'static [IterState] {
        &[
            IterState::A, IterState::B, IterState::C,
            IterState::D, IterState::E, IterState::F,
            IterState::G, IterState::H, IterState::I,
            IterState::J, IterState::K, IterState::L,
        ]
    }

    /// Get iteration index (0-11)
    pub fn index(&self) -> usize {
        match self {
            IterState::A => 0,
            IterState::B => 1,
            IterState::C => 2,
            IterState::D => 3,
            IterState::E => 4,
            IterState::F => 5,
            IterState::G => 6,
            IterState::H => 7,
            IterState::I => 8,
            IterState::J => 9,
            IterState::K => 10,
            IterState::L => 11,
        }
    }

    /// Get iteration from index
    pub fn from_index(idx: usize) -> Option<IterState> {
        match idx {
            0 => Some(IterState::A),
            1 => Some(IterState::B),
            2 => Some(IterState::C),
            3 => Some(IterState::D),
            4 => Some(IterState::E),
            5 => Some(IterState::F),
            6 => Some(IterState::G),
            7 => Some(IterState::H),
            8 => Some(IterState::I),
            9 => Some(IterState::J),
            10 => Some(IterState::K),
            11 => Some(IterState::L),
            _ => None,
        }
    }

    /// Get phase offset in radians for this iteration state
    pub fn phase_offset(&self) -> f64 {
        let degrees = match self {
            IterState::A => 0.0,
            IterState::B => 10.0,
            IterState::C => 20.0,
            IterState::D => 40.0,
            IterState::E => 50.0,
            IterState::F => 60.0,
            IterState::G => 80.0,
            IterState::H => 90.0,
            IterState::I => 100.0,
            IterState::J => 120.0,
            IterState::K => 135.0,
            IterState::L => 150.0,
        };
        degrees * PI / 180.0
    }

    /// Get analysis depth (0.0 = coarse, 1.0 = deep)
    pub fn depth(&self) -> f64 {
        self.index() as f64 / 11.0
    }
}

/// Variant - A Domain Specialist
///
/// Each variant has:
/// - A specific domain (Kernel, Filesystem, etc.)
/// - An iteration state (coarse to fine)
/// - A unique representation
/// - Phase synchronized to Being
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Variant {
    /// Unique identifier
    pub id: usize,
    
    /// Domain specialization
    pub domain: Domain,
    
    /// Current phase (synchronized to Being)
    pub local_phase: f64,
    
    /// Phase offset based on iteration state
    pub phase_offset: f64,
    
    /// Domain-specific representation/embedding
    pub representation: Vec<f64>,
    
    /// Iteration state (analysis depth)
    pub iteration_state: IterState,
    
    /// Local field contribution
    pub local_field: Vec<f64>,
    
    /// How strongly this variant is resonating
    pub resonance_strength: f64,
    
    /// Natural frequency (domain-specific)
    pub natural_freq: f64,
    
    /// Mass (importance/confidence)
    pub mass: f64,
    
    /// Position in embedding space
    pub position: Vec<f64>,
    
    /// Velocity in embedding space (for gravity dynamics)
    #[serde(skip)]
    pub velocity: Vec<f64>,
}

impl Variant {
    /// Create a new Variant
    pub fn new(id: usize, domain: Domain, iteration_state: IterState, embed_dim: usize) -> Self {
        let mut rng = rand::thread_rng();
        let normal = Normal::new(0.0, 0.1).unwrap();
        
        // Initialize representation with small random values
        let representation: Vec<f64> = (0..embed_dim)
            .map(|_| normal.sample(&mut rng))
            .collect();
        
        // Initialize position based on domain
        let position: Vec<f64> = (0..embed_dim)
            .map(|i| {
                let domain_offset = domain.index() as f64 * 0.5;
                let iter_offset = iteration_state.index() as f64 * 0.1;
                domain_offset + iter_offset + normal.sample(&mut rng) * 0.1
            })
            .collect();
        
        Self {
            id,
            domain,
            local_phase: rng.gen::<f64>() * TWO_PI,
            phase_offset: iteration_state.phase_offset(),
            representation,
            iteration_state,
            local_field: vec![0.0; embed_dim],
            resonance_strength: 0.0,
            natural_freq: domain.base_frequency() + normal.sample(&mut rng) * 0.1,
            mass: 1.0,
            position,
            velocity: vec![0.0; embed_dim],
        }
    }

    /// Create all variants for a configuration
    pub fn create_all(config: &VariantsConfig) -> Vec<Variant> {
        let mut variants = Vec::with_capacity(config.count);
        let domains = Domain::all();
        let iterations = IterState::all();
        
        let iters_per_domain = config.iterations_per_domain.min(iterations.len());
        
        for (d_idx, domain) in domains.iter().enumerate() {
            for i_idx in 0..iters_per_domain {
                let id = d_idx * iters_per_domain + i_idx;
                if id >= config.count {
                    break;
                }
                
                let iter_state = iterations[i_idx % iterations.len()];
                variants.push(Variant::new(id, *domain, iter_state, config.embed_dim));
            }
        }
        
        variants
    }

    /// Get current state snapshot (for parallel processing)
    pub fn state(&self) -> VariantState {
        VariantState {
            id: self.id,
            phase: self.local_phase,
            position: self.position.clone(),
            mass: self.mass,
            local_field: self.local_field.clone(),
        }
    }

    /// Receive broadcast from Being and process input
    pub fn receive_and_process(&mut self, broadcast: &Broadcast, input: &Input) {
        // 1. Sync phase to Being's rhythm
        self.sync_phase(broadcast.global_rhythm);
        
        // 2. Encode input in domain-specific representation
        let encoded = self.encode_input(input);
        
        // 3. Update local field based on encoding and broadcast
        self.update_local_field(&encoded, broadcast);
        
        // 4. Compute resonance strength
        self.resonance_strength = self.compute_resonance_from_broadcast(broadcast);
    }

    /// Sync phase with Being (Kuramoto-style coupling)
    fn sync_phase(&mut self, master_phase: f64) {
        let coupling = 0.5;
        let target_phase = master_phase + self.phase_offset;
        let phase_diff = target_phase - self.local_phase;
        
        // Kuramoto update
        self.local_phase += coupling * phase_diff.sin();
        self.local_phase = math::wrap_angle(self.local_phase);
    }

    /// Encode input in domain-specific way
    fn encode_input(&self, input: &Input) -> Vec<f64> {
        let mut encoding = vec![0.0; self.representation.len()];
        
        // Check for domain keywords in error message
        let error_lower = input.error.to_lowercase();
        let keywords = self.domain.keywords();
        
        let mut keyword_score = 0.0;
        for keyword in keywords {
            if error_lower.contains(keyword) {
                keyword_score += 1.0;
            }
        }
        
        // Normalize keyword score
        keyword_score = (keyword_score / keywords.len() as f64).min(1.0);
        
        // Apply keyword score to representation
        for (i, r) in self.representation.iter().enumerate() {
            encoding[i] = r * (1.0 + keyword_score * 2.0);
        }
        
        // Add context-based modulation
        let context_factor = self.context_relevance(input);
        for e in &mut encoding {
            *e *= 1.0 + context_factor;
        }
        
        encoding
    }

    /// Compute context relevance for this domain
    fn context_relevance(&self, input: &Input) -> f64 {
        use crate::BootStage;
        
        let stage_relevance = match (self.domain, input.context.boot_stage) {
            (Domain::Boot, BootStage::Bootloader) => 1.0,
            (Domain::Boot, BootStage::Kernel) => 0.8,
            (Domain::Kernel, BootStage::Kernel) => 1.0,
            (Domain::Kernel, BootStage::Init) => 0.7,
            (Domain::Filesystem, BootStage::Init) => 0.9,
            (Domain::Service, BootStage::Services) => 1.0,
            (Domain::Service, BootStage::Ready) => 0.8,
            (Domain::Network, BootStage::Services) => 0.7,
            (Domain::Network, BootStage::Ready) => 0.9,
            _ => 0.3,
        };
        
        let state_relevance = match self.domain {
            Domain::Filesystem => {
                if !input.state.root_mounted { 1.0 } else { 0.3 }
            }
            Domain::Network => {
                if !input.state.network_up { 0.8 } else { 0.2 }
            }
            Domain::Memory => {
                if !input.state.memory_available { 1.0 } else { 0.2 }
            }
            _ => 0.5,
        };
        
        (stage_relevance + state_relevance) / 2.0
    }

    /// Update local field based on encoding and broadcast
    fn update_local_field(&mut self, encoded: &[f64], broadcast: &Broadcast) {
        let depth = self.iteration_state.depth();
        
        for (i, e) in encoded.iter().enumerate() {
            if i < self.local_field.len() {
                // Combine encoding with broadcast gradient
                let gradient_influence = if i < broadcast.field_gradient.len() {
                    broadcast.field_gradient[i]
                } else {
                    0.0
                };
                
                // Deeper iterations have more refined fields
                let refinement = 1.0 + depth * 0.5;
                
                self.local_field[i] = e * refinement + gradient_influence * 0.3;
            }
        }
    }

    /// Compute resonance strength from broadcast
    fn compute_resonance_from_broadcast(&self, broadcast: &Broadcast) -> f64 {
        // Phase alignment with Being
        let phase_alignment = (self.local_phase - broadcast.global_rhythm).cos();
        
        // Field alignment
        let field_dot = math::dot(&self.local_field, &broadcast.attractor_state);
        let field_mag = math::magnitude(&self.local_field) * math::magnitude(&broadcast.attractor_state);
        let field_alignment = if field_mag > 1e-10 {
            field_dot / field_mag
        } else {
            0.0
        };
        
        // Combine alignments
        let resonance = 0.6 * (phase_alignment + 1.0) / 2.0 + 0.4 * (field_alignment + 1.0) / 2.0;
        
        resonance.max(0.0).min(1.0)
    }

    /// Compute resonance with other variants (for field settling)
    pub fn compute_resonance(&self, others: &[VariantState]) -> f64 {
        if others.is_empty() {
            return 0.0;
        }
        
        let mut total_resonance = 0.0;
        let mut count = 0;
        
        for other in others {
            if other.id == self.id {
                continue;
            }
            
            // Phase resonance
            let phase_diff = self.local_phase - other.phase;
            let phase_resonance = phase_diff.cos();
            
            // Position proximity
            let dist = math::distance(&self.position, &other.position);
            let proximity = 1.0 / (1.0 + dist);
            
            // Mass-weighted
            let weight = other.mass;
            
            total_resonance += weight * (0.7 * phase_resonance + 0.3 * proximity);
            count += 1;
        }
        
        if count > 0 {
            total_resonance / count as f64
        } else {
            0.0
        }
    }

    /// Resonate with another variant (mutual influence)
    pub fn resonate_with(&mut self, other: &VariantState, coupling: f64) {
        // Kuramoto phase coupling
        let phase_diff = other.phase - self.local_phase;
        self.local_phase += coupling * phase_diff.sin();
        self.local_phase = math::wrap_angle(self.local_phase);
        
        // Field coupling - exchange information
        let field_len = self.local_field.len().min(other.local_field.len());
        for i in 0..field_len {
            let diff = other.local_field[i] - self.local_field[i];
            self.local_field[i] += coupling * 0.1 * diff;
        }
    }

    /// Update position based on gravity (called during settling)
    pub fn apply_gravity(&mut self, force: &[f64], damping: f64, dt: f64) {
        // Update velocity
        for (i, f) in force.iter().enumerate() {
            if i < self.velocity.len() {
                self.velocity[i] += f * dt;
                self.velocity[i] *= 1.0 - damping;
            }
        }
        
        // Update position
        for (i, v) in self.velocity.iter().enumerate() {
            if i < self.position.len() {
                self.position[i] += v * dt;
            }
        }
    }

    /// Update mass based on resonance
    pub fn update_mass(&mut self, resonance: f64, dt: f64) {
        let growth_rate = 0.1;
        let decay_rate = 0.01;
        
        self.mass += (growth_rate * resonance - decay_rate * self.mass) * dt;
        self.mass = self.mass.clamp(0.1, 10.0);
    }

    /// Contribution to output (for learning)
    pub fn contribution_to_output(&self, _output: &crate::Output) -> f64 {
        // Contribution based on resonance and mass
        self.resonance_strength * self.mass / 10.0
    }

    /// Reset variant to initial state
    pub fn reset(&mut self) {
        let mut rng = rand::thread_rng();
        self.local_phase = rng.gen::<f64>() * TWO_PI;
        self.local_field.fill(0.0);
        self.resonance_strength = 0.0;
        self.mass = 1.0;
        self.velocity.fill(0.0);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_domain_keywords() {
        let kernel = Domain::Kernel;
        assert!(kernel.keywords().contains(&"panic"));
        
        let fs = Domain::Filesystem;
        assert!(fs.keywords().contains(&"mount"));
    }

    #[test]
    fn test_iter_state_phase_offset() {
        assert_eq!(IterState::A.phase_offset(), 0.0);
        assert!(IterState::L.phase_offset() > IterState::A.phase_offset());
    }

    #[test]
    fn test_variant_creation() {
        let v = Variant::new(0, Domain::Kernel, IterState::A, 64);
        
        assert_eq!(v.id, 0);
        assert_eq!(v.domain, Domain::Kernel);
        assert_eq!(v.iteration_state, IterState::A);
        assert_eq!(v.representation.len(), 64);
    }

    #[test]
    fn test_create_all_variants() {
        let config = VariantsConfig {
            count: 64,
            domains: 8,
            iterations_per_domain: 8,
            embed_dim: 64,
        };
        
        let variants = Variant::create_all(&config);
        assert_eq!(variants.len(), 64);
    }

    #[test]
    fn test_variant_state() {
        let v = Variant::new(0, Domain::Kernel, IterState::A, 64);
        let state = v.state();
        
        assert_eq!(state.id, 0);
        assert_eq!(state.position.len(), 64);
    }
}
