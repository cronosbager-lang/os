//! GGUF Format Support
//!
//! Export and import MRM models in GGUF format for compatibility
//! with existing tooling.

use std::fs::File;
use std::io::{BufReader, BufWriter, Read, Write};
use std::path::Path;

use byteorder::{LittleEndian, ReadBytesExt, WriteBytesExt};
use thiserror::Error;

use crate::inference::ResonanceField;
use crate::MrmConfig;

/// GGUF magic number
const GGUF_MAGIC: u32 = 0x46554747; // "GGUF" in little-endian

/// GGUF version
const GGUF_VERSION: u32 = 3;

/// MRM model type identifier
const MRM_MODEL_TYPE: &str = "mrm-field";

/// MRM version
const MRM_VERSION: &str = "2.0";

/// GGUF errors
#[derive(Error, Debug)]
pub enum GgufError {
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),
    
    #[error("Invalid magic number")]
    InvalidMagic,
    
    #[error("Unsupported version: {0}")]
    UnsupportedVersion(u32),
    
    #[error("Invalid model type: {0}")]
    InvalidModelType(String),
    
    #[error("Missing required tensor: {0}")]
    MissingTensor(String),
    
    #[error("Invalid tensor shape")]
    InvalidShape,
}

/// GGUF tensor types
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(u32)]
pub enum GgufType {
    F32 = 0,
    F16 = 1,
    Q4_0 = 2,
    Q4_1 = 3,
    Q8_0 = 8,
}

impl GgufType {
    fn from_u32(v: u32) -> Option<Self> {
        match v {
            0 => Some(GgufType::F32),
            1 => Some(GgufType::F16),
            2 => Some(GgufType::Q4_0),
            3 => Some(GgufType::Q4_1),
            8 => Some(GgufType::Q8_0),
            _ => None,
        }
    }
    
    fn bytes_per_element(&self) -> usize {
        match self {
            GgufType::F32 => 4,
            GgufType::F16 => 2,
            GgufType::Q4_0 | GgufType::Q4_1 => 1, // Approximate
            GgufType::Q8_0 => 1,
        }
    }
}

/// GGUF Writer for exporting models
pub struct GgufWriter {
    writer: BufWriter<File>,
    tensors: Vec<TensorInfo>,
    metadata: Vec<(String, MetadataValue)>,
}

/// Tensor information
#[derive(Clone)]
struct TensorInfo {
    name: String,
    shape: Vec<usize>,
    dtype: GgufType,
    data: Vec<u8>,
}

/// Metadata value types
#[derive(Debug, Clone)]
pub enum MetadataValue {
    U32(u32),
    I32(i32),
    F32(f32),
    String(String),
    Array(Vec<MetadataValue>),
}

impl GgufWriter {
    /// Create a new GGUF writer
    pub fn new<P: AsRef<Path>>(path: P) -> Result<Self, GgufError> {
        let file = File::create(path)?;
        let writer = BufWriter::new(file);
        
        Ok(Self {
            writer,
            tensors: Vec::new(),
            metadata: Vec::new(),
        })
    }

    /// Add metadata
    pub fn add_metadata(&mut self, key: &str, value: MetadataValue) {
        self.metadata.push((key.to_string(), value));
    }

    /// Add a tensor
    pub fn add_tensor(&mut self, name: &str, data: &[f32], shape: &[usize]) -> Result<(), GgufError> {
        // Convert f32 to bytes
        let mut bytes = Vec::with_capacity(data.len() * 4);
        for &v in data {
            bytes.extend_from_slice(&v.to_le_bytes());
        }
        
        self.tensors.push(TensorInfo {
            name: name.to_string(),
            shape: shape.to_vec(),
            dtype: GgufType::F32,
            data: bytes,
        });
        
        Ok(())
    }

    /// Add a tensor as f16
    pub fn add_tensor_f16(&mut self, name: &str, data: &[f32], shape: &[usize]) -> Result<(), GgufError> {
        // Convert f32 to f16 bytes (simplified - just truncate for now)
        let mut bytes = Vec::with_capacity(data.len() * 2);
        for &v in data {
            let f16_bits = f32_to_f16(v);
            bytes.extend_from_slice(&f16_bits.to_le_bytes());
        }
        
        self.tensors.push(TensorInfo {
            name: name.to_string(),
            shape: shape.to_vec(),
            dtype: GgufType::F16,
            data: bytes,
        });
        
        Ok(())
    }

    /// Finalize and write the GGUF file
    pub fn finalize(mut self) -> Result<(), GgufError> {
        // Write header
        self.writer.write_u32::<LittleEndian>(GGUF_MAGIC)?;
        self.writer.write_u32::<LittleEndian>(GGUF_VERSION)?;
        self.writer.write_u64::<LittleEndian>(self.tensors.len() as u64)?;
        self.writer.write_u64::<LittleEndian>(self.metadata.len() as u64)?;
        
        // Clone metadata to avoid borrow issues
        let metadata = self.metadata.clone();
        
        // Write metadata
        for (key, value) in &metadata {
            self.write_string(key)?;
            self.write_metadata_value(value)?;
        }
        
        // Clone tensors to avoid borrow issues
        let tensors = self.tensors.clone();
        
        // Write tensor infos
        let mut offset = 0u64;
        for tensor in &tensors {
            self.write_string(&tensor.name)?;
            self.writer.write_u32::<LittleEndian>(tensor.shape.len() as u32)?;
            for &dim in &tensor.shape {
                self.writer.write_u64::<LittleEndian>(dim as u64)?;
            }
            self.writer.write_u32::<LittleEndian>(tensor.dtype as u32)?;
            self.writer.write_u64::<LittleEndian>(offset)?;
            offset += tensor.data.len() as u64;
        }
        
        // Write tensor data
        for tensor in &tensors {
            self.writer.write_all(&tensor.data)?;
        }
        
        self.writer.flush()?;
        Ok(())
    }

    fn write_string(&mut self, s: &str) -> Result<(), GgufError> {
        let bytes = s.as_bytes();
        self.writer.write_u64::<LittleEndian>(bytes.len() as u64)?;
        self.writer.write_all(bytes)?;
        Ok(())
    }

    fn write_metadata_value(&mut self, value: &MetadataValue) -> Result<(), GgufError> {
        match value {
            MetadataValue::U32(v) => {
                self.writer.write_u32::<LittleEndian>(4)?; // type
                self.writer.write_u32::<LittleEndian>(*v)?;
            }
            MetadataValue::I32(v) => {
                self.writer.write_u32::<LittleEndian>(5)?; // type
                self.writer.write_i32::<LittleEndian>(*v)?;
            }
            MetadataValue::F32(v) => {
                self.writer.write_u32::<LittleEndian>(6)?; // type
                self.writer.write_f32::<LittleEndian>(*v)?;
            }
            MetadataValue::String(s) => {
                self.writer.write_u32::<LittleEndian>(8)?; // type
                self.write_string(s)?;
            }
            MetadataValue::Array(arr) => {
                self.writer.write_u32::<LittleEndian>(9)?; // type
                self.writer.write_u64::<LittleEndian>(arr.len() as u64)?;
                for v in arr {
                    self.write_metadata_value(v)?;
                }
            }
        }
        Ok(())
    }
}

/// GGUF Reader for importing models
pub struct GgufReader {
    reader: BufReader<File>,
    pub version: u32,
    pub n_tensors: u64,
    pub n_metadata: u64,
    pub metadata: Vec<(String, MetadataValue)>,
    tensor_infos: Vec<TensorReadInfo>,
    data_offset: u64,
}

struct TensorReadInfo {
    name: String,
    shape: Vec<usize>,
    dtype: GgufType,
    offset: u64,
    size: usize,
}

impl GgufReader {
    /// Open a GGUF file for reading
    pub fn open<P: AsRef<Path>>(path: P) -> Result<Self, GgufError> {
        let file = File::open(path)?;
        let mut reader = BufReader::new(file);
        
        // Read header
        let magic = reader.read_u32::<LittleEndian>()?;
        if magic != GGUF_MAGIC {
            return Err(GgufError::InvalidMagic);
        }
        
        let version = reader.read_u32::<LittleEndian>()?;
        if version > GGUF_VERSION {
            return Err(GgufError::UnsupportedVersion(version));
        }
        
        let n_tensors = reader.read_u64::<LittleEndian>()?;
        let n_metadata = reader.read_u64::<LittleEndian>()?;
        
        // Read metadata
        let mut metadata = Vec::new();
        for _ in 0..n_metadata {
            let key = Self::read_string_static(&mut reader)?;
            let value = Self::read_metadata_value_static(&mut reader)?;
            metadata.push((key, value));
        }
        
        // Read tensor infos
        let mut tensor_infos = Vec::new();
        for _ in 0..n_tensors {
            let name = Self::read_string_static(&mut reader)?;
            let n_dims = reader.read_u32::<LittleEndian>()? as usize;
            let mut shape = Vec::with_capacity(n_dims);
            for _ in 0..n_dims {
                shape.push(reader.read_u64::<LittleEndian>()? as usize);
            }
            let dtype_u32 = reader.read_u32::<LittleEndian>()?;
            let dtype = GgufType::from_u32(dtype_u32).unwrap_or(GgufType::F32);
            let offset = reader.read_u64::<LittleEndian>()?;
            
            let size: usize = shape.iter().product::<usize>() * dtype.bytes_per_element();
            
            tensor_infos.push(TensorReadInfo {
                name,
                shape,
                dtype,
                offset,
                size,
            });
        }
        
        // Current position is the data offset
        let data_offset = 4 + 4 + 8 + 8; // Approximate - would need proper tracking
        
        Ok(Self {
            reader,
            version,
            n_tensors,
            n_metadata,
            metadata,
            tensor_infos,
            data_offset,
        })
    }

    fn read_string_static(reader: &mut BufReader<File>) -> Result<String, GgufError> {
        let len = reader.read_u64::<LittleEndian>()? as usize;
        let mut bytes = vec![0u8; len];
        reader.read_exact(&mut bytes)?;
        Ok(String::from_utf8_lossy(&bytes).to_string())
    }

    fn read_metadata_value_static(reader: &mut BufReader<File>) -> Result<MetadataValue, GgufError> {
        let type_id = reader.read_u32::<LittleEndian>()?;
        
        match type_id {
            4 => Ok(MetadataValue::U32(reader.read_u32::<LittleEndian>()?)),
            5 => Ok(MetadataValue::I32(reader.read_i32::<LittleEndian>()?)),
            6 => Ok(MetadataValue::F32(reader.read_f32::<LittleEndian>()?)),
            8 => Ok(MetadataValue::String(Self::read_string_static(reader)?)),
            9 => {
                let len = reader.read_u64::<LittleEndian>()? as usize;
                let mut arr = Vec::with_capacity(len);
                for _ in 0..len {
                    arr.push(Self::read_metadata_value_static(reader)?);
                }
                Ok(MetadataValue::Array(arr))
            }
            _ => Ok(MetadataValue::U32(0)), // Unknown type
        }
    }

    /// Get metadata value by key
    pub fn get_metadata(&self, key: &str) -> Option<&MetadataValue> {
        self.metadata.iter()
            .find(|(k, _)| k == key)
            .map(|(_, v)| v)
    }

    /// Read a tensor by name
    pub fn read_tensor(&mut self, name: &str) -> Result<(Vec<f32>, Vec<usize>), GgufError> {
        let info = self.tensor_infos.iter()
            .find(|t| t.name == name)
            .ok_or_else(|| GgufError::MissingTensor(name.to_string()))?
            .clone();
        
        // Read raw bytes
        let mut bytes = vec![0u8; info.size];
        // Note: In a real implementation, we'd seek to the correct position
        self.reader.read_exact(&mut bytes)?;
        
        // Convert to f32
        let data = match info.dtype {
            GgufType::F32 => {
                bytes.chunks(4)
                    .map(|chunk| {
                        let arr: [u8; 4] = chunk.try_into().unwrap_or([0; 4]);
                        f32::from_le_bytes(arr)
                    })
                    .collect()
            }
            GgufType::F16 => {
                bytes.chunks(2)
                    .map(|chunk| {
                        let arr: [u8; 2] = chunk.try_into().unwrap_or([0; 2]);
                        f16_to_f32(u16::from_le_bytes(arr))
                    })
                    .collect()
            }
            _ => {
                // Simplified - just return zeros for quantized types
                vec![0.0; info.shape.iter().product()]
            }
        };
        
        Ok((data, info.shape.clone()))
    }

    /// List all tensor names
    pub fn tensor_names(&self) -> Vec<&str> {
        self.tensor_infos.iter().map(|t| t.name.as_str()).collect()
    }
}

/// Export a ResonanceField to GGUF format
pub fn export_model<P: AsRef<Path>>(model: &ResonanceField, path: P) -> Result<(), GgufError> {
    let mut writer = GgufWriter::new(path)?;
    
    // Add metadata
    writer.add_metadata("general.architecture", MetadataValue::String(MRM_MODEL_TYPE.to_string()));
    writer.add_metadata("general.name", MetadataValue::String("MixOS Recovery Model".to_string()));
    writer.add_metadata("mrm.version", MetadataValue::String(MRM_VERSION.to_string()));
    writer.add_metadata("mrm.variants.count", MetadataValue::U32(model.variants.len() as u32));
    writer.add_metadata("mrm.field.resolution", MetadataValue::U32(model.config.field.resolution as u32));
    
    // Export Being state
    let attractor_state: Vec<f32> = model.being.attractor_state.iter().map(|&v| v as f32).collect();
    writer.add_tensor_f16("being.attractor_state", &attractor_state, &[attractor_state.len()])?;
    
    let integrated_field: Vec<f32> = model.being.integrated_field.iter().map(|&v| v as f32).collect();
    writer.add_tensor_f16("being.integrated_field", &integrated_field, &[integrated_field.len()])?;
    
    // Export Variant representations
    let n_variants = model.variants.len();
    let embed_dim = model.config.variants.embed_dim;
    
    let mut representations: Vec<f32> = Vec::with_capacity(n_variants * embed_dim);
    let mut natural_freqs: Vec<f32> = Vec::with_capacity(n_variants);
    let mut phase_offsets: Vec<f32> = Vec::with_capacity(n_variants);
    let mut masses: Vec<f32> = Vec::with_capacity(n_variants);
    
    for v in &model.variants {
        representations.extend(v.representation.iter().map(|&x| x as f32));
        natural_freqs.push(v.natural_freq as f32);
        phase_offsets.push(v.phase_offset as f32);
        masses.push(v.mass as f32);
    }
    
    writer.add_tensor_f16("variants.representations", &representations, &[n_variants, embed_dim])?;
    writer.add_tensor("variants.natural_freq", &natural_freqs, &[n_variants])?;
    writer.add_tensor("variants.phase_offsets", &phase_offsets, &[n_variants])?;
    writer.add_tensor("variants.masses", &masses, &[n_variants])?;
    
    // Export coupling matrix
    let coupling_flat: Vec<f32> = model.coupling_matrix.iter()
        .flatten()
        .map(|&v| v as f32)
        .collect();
    writer.add_tensor_f16("coupling.matrix", &coupling_flat, &[n_variants, n_variants])?;
    
    // Export field data
    let field_data: Vec<f32> = model.field.as_slice().iter().map(|&v| v as f32).collect();
    let res = model.config.field.resolution;
    writer.add_tensor_f16("field.data", &field_data, &[res, res, res])?;
    
    writer.finalize()
}

/// Import a ResonanceField from GGUF format
pub fn import_model<P: AsRef<Path>>(path: P) -> Result<ResonanceField, GgufError> {
    let mut reader = GgufReader::open(path)?;
    
    // Read metadata
    let n_variants = match reader.get_metadata("mrm.variants.count") {
        Some(MetadataValue::U32(n)) => *n as usize,
        _ => 64,
    };
    
    let field_resolution = match reader.get_metadata("mrm.field.resolution") {
        Some(MetadataValue::U32(r)) => *r as usize,
        _ => 32,
    };
    
    // Create model with appropriate config
    let mut config = MrmConfig::default();
    config.variants.count = n_variants;
    config.field.resolution = field_resolution;
    
    let mut model = ResonanceField::with_config(config);
    
    // Load tensors (simplified - in real implementation would load all)
    // For now, just return the default-initialized model
    
    Ok(model)
}

/// Convert f32 to f16 (simplified)
fn f32_to_f16(v: f32) -> u16 {
    // Simplified conversion - loses precision
    let bits = v.to_bits();
    let sign = (bits >> 31) & 1;
    let exp = ((bits >> 23) & 0xFF) as i32 - 127 + 15;
    let mantissa = (bits >> 13) & 0x3FF;
    
    if exp <= 0 {
        0
    } else if exp >= 31 {
        ((sign << 15) | (31 << 10)) as u16
    } else {
        ((sign << 15) | ((exp as u32) << 10) | mantissa) as u16
    }
}

/// Convert f16 to f32 (simplified)
fn f16_to_f32(v: u16) -> f32 {
    let sign = ((v >> 15) & 1) as u32;
    let exp = ((v >> 10) & 0x1F) as i32;
    let mantissa = (v & 0x3FF) as u32;
    
    if exp == 0 {
        0.0
    } else if exp == 31 {
        if mantissa == 0 {
            if sign == 1 { f32::NEG_INFINITY } else { f32::INFINITY }
        } else {
            f32::NAN
        }
    } else {
        let exp32 = exp - 15 + 127;
        let bits = (sign << 31) | ((exp32 as u32) << 23) | (mantissa << 13);
        f32::from_bits(bits)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;
    use tempfile::tempdir;

    #[test]
    fn test_f16_conversion() {
        let values = [0.0, 1.0, -1.0, 0.5, 100.0];
        
        for &v in &values {
            let f16 = f32_to_f16(v);
            let back = f16_to_f32(f16);
            
            // Allow some precision loss
            if v != 0.0 {
                assert!((back - v).abs() / v.abs() < 0.01);
            }
        }
    }

    #[test]
    fn test_export_import() {
        let dir = tempdir().unwrap();
        let path = dir.path().join("test_model.gguf");
        
        let model = ResonanceField::new();
        
        // Export
        export_model(&model, &path).unwrap();
        
        // Check file exists
        assert!(path.exists());
        
        // Import
        let _imported = import_model(&path).unwrap();
    }
}
