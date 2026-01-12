//! Field3D - 3D Field Topology
//!
//! Represents the continuous field in which variants exist and interact.
//! Information propagates through this field via diffusion.

use serde::{Deserialize, Serialize};

/// 3D Field structure for information propagation
#[derive(Debug, Clone)]
pub struct Field3D {
    /// Field resolution (size in each dimension)
    pub resolution: usize,
    
    /// Field data (flattened 3D array)
    pub data: Vec<f64>,
    
    /// Gradient cache
    gradient_cache: Vec<f64>,
    
    /// Whether gradient cache is valid
    gradient_valid: bool,
}

impl Field3D {
    /// Create a new Field3D with given resolution
    pub fn new(resolution: usize) -> Self {
        let size = resolution * resolution * resolution;
        Self {
            resolution,
            data: vec![0.0; size],
            gradient_cache: vec![0.0; size],
            gradient_valid: false,
        }
    }

    /// Get the total size of the field
    pub fn size(&self) -> usize {
        self.data.len()
    }

    /// Convert 3D coordinates to linear index
    #[inline]
    pub fn index(&self, x: usize, y: usize, z: usize) -> usize {
        let r = self.resolution;
        z * r * r + y * r + x
    }

    /// Convert linear index to 3D coordinates
    #[inline]
    pub fn coords(&self, idx: usize) -> (usize, usize, usize) {
        let r = self.resolution;
        let z = idx / (r * r);
        let rem = idx % (r * r);
        let y = rem / r;
        let x = rem % r;
        (x, y, z)
    }

    /// Get value at 3D coordinates
    pub fn get(&self, x: usize, y: usize, z: usize) -> f64 {
        let idx = self.index(x, y, z);
        self.data.get(idx).copied().unwrap_or(0.0)
    }

    /// Set value at 3D coordinates
    pub fn set(&mut self, x: usize, y: usize, z: usize, value: f64) {
        let idx = self.index(x, y, z);
        if idx < self.data.len() {
            self.data[idx] = value;
            self.gradient_valid = false;
        }
    }

    /// Get value at linear index
    pub fn get_linear(&self, idx: usize) -> f64 {
        self.data.get(idx).copied().unwrap_or(0.0)
    }

    /// Set value at linear index
    pub fn set_linear(&mut self, idx: usize, value: f64) {
        if idx < self.data.len() {
            self.data[idx] = value;
            self.gradient_valid = false;
        }
    }

    /// Clear the field (set all values to zero)
    pub fn clear(&mut self) {
        self.data.fill(0.0);
        self.gradient_valid = false;
    }

    /// Inject values into the field (from integrated field)
    pub fn inject_values(&mut self, values: &[f64]) {
        // Distribute values across the field
        let r = self.resolution;
        
        for (i, &v) in values.iter().enumerate() {
            // Map value index to field position
            let x = (i * r / values.len().max(1)) % r;
            let y = (i * 7) % r;  // Spread across y
            let z = (i * 13) % r; // Spread across z
            
            let idx = self.index(x, y, z);
            if idx < self.data.len() {
                self.data[idx] += v;
            }
        }
        
        self.gradient_valid = false;
    }

    /// Compute the gradient of the field
    pub fn gradient(&self) -> Vec<f64> {
        if self.gradient_valid {
            return self.gradient_cache.clone();
        }
        
        let r = self.resolution;
        let mut grad = vec![0.0; self.data.len()];
        
        for z in 0..r {
            for y in 0..r {
                for x in 0..r {
                    let idx = self.index(x, y, z);
                    let center = self.data[idx];
                    
                    // Compute gradient magnitude using central differences
                    let dx = if x > 0 && x < r - 1 {
                        (self.get(x + 1, y, z) - self.get(x - 1, y, z)) / 2.0
                    } else {
                        0.0
                    };
                    
                    let dy = if y > 0 && y < r - 1 {
                        (self.get(x, y + 1, z) - self.get(x, y - 1, z)) / 2.0
                    } else {
                        0.0
                    };
                    
                    let dz = if z > 0 && z < r - 1 {
                        (self.get(x, y, z + 1) - self.get(x, y, z - 1)) / 2.0
                    } else {
                        0.0
                    };
                    
                    grad[idx] = (dx * dx + dy * dy + dz * dz).sqrt();
                }
            }
        }
        
        grad
    }

    /// Apply diffusion to the field
    pub fn diffuse(&mut self, diffusion_rate: f64, dt: f64) {
        let r = self.resolution;
        let mut new_data = self.data.clone();
        
        let alpha = diffusion_rate * dt;
        
        for z in 1..r-1 {
            for y in 1..r-1 {
                for x in 1..r-1 {
                    let idx = self.index(x, y, z);
                    let center = self.data[idx];
                    
                    // 6-point stencil Laplacian
                    let laplacian = 
                        self.get(x + 1, y, z) + self.get(x - 1, y, z) +
                        self.get(x, y + 1, z) + self.get(x, y - 1, z) +
                        self.get(x, y, z + 1) + self.get(x, y, z - 1) -
                        6.0 * center;
                    
                    new_data[idx] = center + alpha * laplacian;
                }
            }
        }
        
        self.data = new_data;
        self.gradient_valid = false;
    }

    /// Apply damping to the field
    pub fn damp(&mut self, damping: f64) {
        for v in &mut self.data {
            *v *= 1.0 - damping;
        }
        self.gradient_valid = false;
    }

    /// Compute the total energy of the field
    pub fn energy(&self) -> f64 {
        // Energy = sum of squared values (like L2 norm)
        self.data.iter().map(|v| v * v).sum::<f64>().sqrt()
    }

    /// Compute the smoothness of the field (lower = smoother)
    pub fn smoothness(&self) -> f64 {
        let grad = self.gradient();
        grad.iter().map(|g| g * g).sum::<f64>().sqrt()
    }

    /// Find local maxima (potential attractors)
    pub fn find_maxima(&self) -> Vec<(usize, usize, usize, f64)> {
        let r = self.resolution;
        let mut maxima = Vec::new();
        
        for z in 1..r-1 {
            for y in 1..r-1 {
                for x in 1..r-1 {
                    let center = self.get(x, y, z);
                    
                    // Check if center is greater than all neighbors
                    let is_max = 
                        center > self.get(x + 1, y, z) &&
                        center > self.get(x - 1, y, z) &&
                        center > self.get(x, y + 1, z) &&
                        center > self.get(x, y - 1, z) &&
                        center > self.get(x, y, z + 1) &&
                        center > self.get(x, y, z - 1);
                    
                    if is_max && center > 0.1 {
                        maxima.push((x, y, z, center));
                    }
                }
            }
        }
        
        // Sort by value (descending)
        maxima.sort_by(|a, b| b.3.partial_cmp(&a.3).unwrap());
        
        maxima
    }

    /// Add a source at a position (inject energy)
    pub fn add_source(&mut self, x: usize, y: usize, z: usize, strength: f64) {
        let idx = self.index(x, y, z);
        if idx < self.data.len() {
            self.data[idx] += strength;
            self.gradient_valid = false;
        }
    }

    /// Add a Gaussian source centered at a position
    pub fn add_gaussian_source(&mut self, cx: f64, cy: f64, cz: f64, strength: f64, sigma: f64) {
        let r = self.resolution;
        let sigma_sq = sigma * sigma;
        
        for z in 0..r {
            for y in 0..r {
                for x in 0..r {
                    let dx = x as f64 - cx;
                    let dy = y as f64 - cy;
                    let dz = z as f64 - cz;
                    let dist_sq = dx * dx + dy * dy + dz * dz;
                    
                    let value = strength * (-dist_sq / (2.0 * sigma_sq)).exp();
                    
                    let idx = self.index(x, y, z);
                    self.data[idx] += value;
                }
            }
        }
        
        self.gradient_valid = false;
    }

    /// Sample the field at a continuous position (trilinear interpolation)
    pub fn sample(&self, x: f64, y: f64, z: f64) -> f64 {
        let r = self.resolution as f64;
        
        // Clamp to valid range
        let x = x.max(0.0).min(r - 1.001);
        let y = y.max(0.0).min(r - 1.001);
        let z = z.max(0.0).min(r - 1.001);
        
        let x0 = x.floor() as usize;
        let y0 = y.floor() as usize;
        let z0 = z.floor() as usize;
        let x1 = (x0 + 1).min(self.resolution - 1);
        let y1 = (y0 + 1).min(self.resolution - 1);
        let z1 = (z0 + 1).min(self.resolution - 1);
        
        let xd = x - x0 as f64;
        let yd = y - y0 as f64;
        let zd = z - z0 as f64;
        
        // Trilinear interpolation
        let c000 = self.get(x0, y0, z0);
        let c001 = self.get(x0, y0, z1);
        let c010 = self.get(x0, y1, z0);
        let c011 = self.get(x0, y1, z1);
        let c100 = self.get(x1, y0, z0);
        let c101 = self.get(x1, y0, z1);
        let c110 = self.get(x1, y1, z0);
        let c111 = self.get(x1, y1, z1);
        
        let c00 = c000 * (1.0 - xd) + c100 * xd;
        let c01 = c001 * (1.0 - xd) + c101 * xd;
        let c10 = c010 * (1.0 - xd) + c110 * xd;
        let c11 = c011 * (1.0 - xd) + c111 * xd;
        
        let c0 = c00 * (1.0 - yd) + c10 * yd;
        let c1 = c01 * (1.0 - yd) + c11 * yd;
        
        c0 * (1.0 - zd) + c1 * zd
    }

    /// Get a flattened slice of the field for serialization
    pub fn as_slice(&self) -> &[f64] {
        &self.data
    }

    /// Create from a flattened slice
    pub fn from_slice(resolution: usize, data: &[f64]) -> Self {
        let expected_size = resolution * resolution * resolution;
        let mut field_data = vec![0.0; expected_size];
        
        for (i, &v) in data.iter().take(expected_size).enumerate() {
            field_data[i] = v;
        }
        
        Self {
            resolution,
            data: field_data,
            gradient_cache: vec![0.0; expected_size],
            gradient_valid: false,
        }
    }
}

impl Default for Field3D {
    fn default() -> Self {
        Self::new(32)
    }
}

// Implement Serialize manually (skip gradient cache)
impl Serialize for Field3D {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        use serde::ser::SerializeStruct;
        let mut state = serializer.serialize_struct("Field3D", 2)?;
        state.serialize_field("resolution", &self.resolution)?;
        state.serialize_field("data", &self.data)?;
        state.end()
    }
}

// Implement Deserialize manually
impl<'de> Deserialize<'de> for Field3D {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        #[derive(Deserialize)]
        struct Field3DData {
            resolution: usize,
            data: Vec<f64>,
        }
        
        let data = Field3DData::deserialize(deserializer)?;
        Ok(Self::from_slice(data.resolution, &data.data))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_field_creation() {
        let field = Field3D::new(8);
        assert_eq!(field.resolution, 8);
        assert_eq!(field.size(), 8 * 8 * 8);
    }

    #[test]
    fn test_index_coords() {
        let field = Field3D::new(8);
        
        let idx = field.index(3, 4, 5);
        let (x, y, z) = field.coords(idx);
        
        assert_eq!(x, 3);
        assert_eq!(y, 4);
        assert_eq!(z, 5);
    }

    #[test]
    fn test_get_set() {
        let mut field = Field3D::new(8);
        
        field.set(2, 3, 4, 1.5);
        assert_eq!(field.get(2, 3, 4), 1.5);
    }

    #[test]
    fn test_diffusion() {
        let mut field = Field3D::new(8);
        
        // Add a point source
        field.set(4, 4, 4, 10.0);
        
        let initial_energy = field.energy();
        
        // Apply diffusion
        field.diffuse(0.1, 0.1);
        
        // Energy should decrease slightly due to spreading
        let final_energy = field.energy();
        assert!(final_energy <= initial_energy);
    }

    #[test]
    fn test_gaussian_source() {
        let mut field = Field3D::new(16);
        
        field.add_gaussian_source(8.0, 8.0, 8.0, 1.0, 2.0);
        
        // Center should have highest value
        let center = field.get(8, 8, 8);
        let edge = field.get(0, 0, 0);
        
        assert!(center > edge);
    }

    #[test]
    fn test_sample() {
        let mut field = Field3D::new(8);
        
        field.set(2, 2, 2, 1.0);
        field.set(3, 2, 2, 2.0);
        
        // Sample at midpoint should interpolate
        let sampled = field.sample(2.5, 2.0, 2.0);
        assert!((sampled - 1.5).abs() < 0.01);
    }
}
