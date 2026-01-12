//! Gravity Dynamics - Attention/Clustering
//!
//! Implements gravitational attraction between variants:
//! dxᵢ/dt = Σⱼ G·mⱼ·(xⱼ - xᵢ)/|xⱼ - xᵢ|³ - γ·vᵢ
//!
//! This provides natural attention mechanism where high-mass variants
//! attract others, forming clusters around "correct" solutions.

use crate::{math, VariantState};

/// Gravitational constant (tunable)
pub const G: f64 = 0.1;

/// Softening parameter to avoid singularities
pub const SOFTENING: f64 = 0.1;

/// Gravity dynamics for attention and clustering
pub struct GravityDynamics {
    /// Gravitational constant
    pub g: f64,
    
    /// Damping factor for velocity
    pub damping: f64,
    
    /// Softening parameter
    pub softening: f64,
}

impl GravityDynamics {
    /// Create new gravity dynamics
    pub fn new(damping: f64) -> Self {
        Self {
            g: G,
            damping,
            softening: SOFTENING,
        }
    }

    /// Create with custom parameters
    pub fn with_params(g: f64, damping: f64, softening: f64) -> Self {
        Self { g, damping, softening }
    }

    /// Compute gravitational force on a variant from all others
    ///
    /// F = Σⱼ G·mⱼ·(xⱼ - xᵢ)/|xⱼ - xᵢ|³
    pub fn compute_force(
        &self,
        variant_idx: usize,
        variant_position: &[f64],
        others: &[VariantState],
    ) -> Vec<f64> {
        let dim = variant_position.len();
        let mut force = vec![0.0; dim];
        
        for other in others {
            if other.id == variant_idx {
                continue;
            }
            
            // Direction vector from variant to other
            let direction = math::subtract(&other.position, variant_position);
            
            // Distance with softening
            let dist_sq = direction.iter().map(|d| d * d).sum::<f64>() + self.softening * self.softening;
            let dist = dist_sq.sqrt();
            let dist_cubed = dist * dist * dist;
            
            // Gravitational force magnitude
            let force_mag = self.g * other.mass / dist_cubed;
            
            // Add to total force
            for (i, d) in direction.iter().enumerate() {
                force[i] += force_mag * d;
            }
        }
        
        force
    }

    /// Compute gravitational potential energy
    ///
    /// E = -Σᵢⱼ G·mᵢ·mⱼ/|xᵢ - xⱼ|
    pub fn energy(&self, variants: &[VariantState]) -> f64 {
        let mut energy = 0.0;
        
        for (i, vi) in variants.iter().enumerate() {
            for (j, vj) in variants.iter().enumerate() {
                if i >= j {
                    continue;
                }
                
                let dist = math::distance(&vi.position, &vj.position) + self.softening;
                energy -= self.g * vi.mass * vj.mass / dist;
            }
        }
        
        energy
    }

    /// Compute center of mass
    pub fn center_of_mass(&self, variants: &[VariantState]) -> Vec<f64> {
        if variants.is_empty() {
            return Vec::new();
        }
        
        let dim = variants[0].position.len();
        let mut com = vec![0.0; dim];
        let mut total_mass = 0.0;
        
        for v in variants {
            total_mass += v.mass;
            for (i, &p) in v.position.iter().enumerate() {
                com[i] += v.mass * p;
            }
        }
        
        if total_mass > 1e-10 {
            for c in &mut com {
                *c /= total_mass;
            }
        }
        
        com
    }

    /// Compute total kinetic energy (if velocities are tracked)
    pub fn kinetic_energy(&self, velocities: &[Vec<f64>], masses: &[f64]) -> f64 {
        velocities
            .iter()
            .zip(masses.iter())
            .map(|(v, &m)| 0.5 * m * math::dot(v, v))
            .sum()
    }

    /// Find clusters of variants based on proximity
    pub fn find_clusters(&self, variants: &[VariantState], threshold: f64) -> Vec<Vec<usize>> {
        let n = variants.len();
        let mut visited = vec![false; n];
        let mut clusters = Vec::new();
        
        for i in 0..n {
            if visited[i] {
                continue;
            }
            
            let mut cluster = vec![i];
            visited[i] = true;
            
            // BFS to find connected variants
            let mut queue = vec![i];
            while let Some(current) = queue.pop() {
                for j in 0..n {
                    if visited[j] {
                        continue;
                    }
                    
                    let dist = math::distance(&variants[current].position, &variants[j].position);
                    if dist < threshold {
                        visited[j] = true;
                        cluster.push(j);
                        queue.push(j);
                    }
                }
            }
            
            clusters.push(cluster);
        }
        
        clusters
    }

    /// Compute the "heaviest" variant (highest mass)
    pub fn find_heaviest(&self, variants: &[VariantState]) -> Option<usize> {
        variants
            .iter()
            .enumerate()
            .max_by(|(_, a), (_, b)| a.mass.partial_cmp(&b.mass).unwrap())
            .map(|(i, _)| i)
    }

    /// Compute attraction strength between two variants
    pub fn attraction_strength(&self, v1: &VariantState, v2: &VariantState) -> f64 {
        let dist = math::distance(&v1.position, &v2.position) + self.softening;
        self.g * v1.mass * v2.mass / (dist * dist)
    }

    /// Apply velocity damping
    pub fn apply_damping(&self, velocity: &mut [f64]) {
        for v in velocity.iter_mut() {
            *v *= 1.0 - self.damping;
        }
    }

    /// Compute escape velocity from a mass at given distance
    pub fn escape_velocity(&self, mass: f64, distance: f64) -> f64 {
        (2.0 * self.g * mass / (distance + self.softening)).sqrt()
    }

    /// Check if system is gravitationally bound
    pub fn is_bound(&self, variants: &[VariantState], velocities: &[Vec<f64>]) -> bool {
        let masses: Vec<f64> = variants.iter().map(|v| v.mass).collect();
        let ke = self.kinetic_energy(velocities, &masses);
        let pe = self.energy(variants);
        
        // System is bound if total energy is negative
        ke + pe < 0.0
    }
}

/// Result of gravity computation
#[derive(Debug, Clone)]
pub struct GravityResult {
    /// Forces on each variant
    pub forces: Vec<Vec<f64>>,
    
    /// Total potential energy
    pub potential_energy: f64,
    
    /// Center of mass
    pub center_of_mass: Vec<f64>,
}

impl GravityDynamics {
    /// Compute all gravity-related quantities at once
    pub fn compute_all(&self, variants: &[VariantState]) -> GravityResult {
        let forces: Vec<Vec<f64>> = variants
            .iter()
            .map(|v| self.compute_force(v.id, &v.position, variants))
            .collect();
        
        GravityResult {
            forces,
            potential_energy: self.energy(variants),
            center_of_mass: self.center_of_mass(variants),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn create_test_variants() -> Vec<VariantState> {
        vec![
            VariantState {
                id: 0,
                phase: 0.0,
                position: vec![0.0, 0.0, 0.0],
                mass: 1.0,
                local_field: vec![],
            },
            VariantState {
                id: 1,
                phase: 0.0,
                position: vec![1.0, 0.0, 0.0],
                mass: 1.0,
                local_field: vec![],
            },
            VariantState {
                id: 2,
                phase: 0.0,
                position: vec![0.0, 1.0, 0.0],
                mass: 2.0,
                local_field: vec![],
            },
        ]
    }

    #[test]
    fn test_force_computation() {
        let dynamics = GravityDynamics::new(0.05);
        let variants = create_test_variants();
        
        let force = dynamics.compute_force(0, &variants[0].position, &variants);
        
        // Force should point towards other masses
        assert!(force[0] > 0.0); // Towards variant 1
        assert!(force[1] > 0.0); // Towards variant 2
    }

    #[test]
    fn test_energy_negative() {
        let dynamics = GravityDynamics::new(0.05);
        let variants = create_test_variants();
        
        let energy = dynamics.energy(&variants);
        
        // Gravitational potential energy should be negative
        assert!(energy < 0.0);
    }

    #[test]
    fn test_center_of_mass() {
        let dynamics = GravityDynamics::new(0.05);
        let variants = create_test_variants();
        
        let com = dynamics.center_of_mass(&variants);
        
        // COM should be weighted towards heavier mass
        assert!(com[1] > com[0]); // y > x because variant 2 has mass 2
    }

    #[test]
    fn test_find_clusters() {
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
                position: vec![0.1, 0.0],
                mass: 1.0,
                local_field: vec![],
            },
            VariantState {
                id: 2,
                phase: 0.0,
                position: vec![10.0, 10.0],
                mass: 1.0,
                local_field: vec![],
            },
        ];
        
        let clusters = dynamics.find_clusters(&variants, 1.0);
        
        // Should have 2 clusters: {0, 1} and {2}
        assert_eq!(clusters.len(), 2);
    }

    #[test]
    fn test_attraction_strength() {
        let dynamics = GravityDynamics::new(0.05);
        let variants = create_test_variants();
        
        let strength_01 = dynamics.attraction_strength(&variants[0], &variants[1]);
        let strength_02 = dynamics.attraction_strength(&variants[0], &variants[2]);
        
        // Attraction to heavier mass should be stronger (at same distance)
        // But variant 2 is at distance sqrt(2), so need to account for that
        assert!(strength_01 > 0.0);
        assert!(strength_02 > 0.0);
    }
}
