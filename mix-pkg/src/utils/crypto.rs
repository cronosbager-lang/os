// Cryptographic utilities for mix-pkg
use anyhow::{Result, Context, bail};
use sha2::{Sha256, Sha512, Digest};
use std::path::Path;
use std::fs::File;
use std::io::{Read, BufReader};

/// Hash algorithm types
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum HashAlgorithm {
    Sha256,
    Sha512,
    Md5,
}

impl HashAlgorithm {
    pub fn from_str(s: &str) -> Option<Self> {
        match s.to_lowercase().as_str() {
            "sha256" => Some(Self::Sha256),
            "sha512" => Some(Self::Sha512),
            "md5" => Some(Self::Md5),
            _ => None,
        }
    }
}

/// Calculate SHA256 hash of a file
pub fn sha256_file(path: &Path) -> Result<String> {
    let file = File::open(path)
        .context(format!("Failed to open file: {:?}", path))?;
    
    let mut reader = BufReader::new(file);
    let mut hasher = Sha256::new();
    let mut buffer = [0u8; 8192];
    
    loop {
        let bytes_read = reader.read(&mut buffer)?;
        if bytes_read == 0 {
            break;
        }
        hasher.update(&buffer[..bytes_read]);
    }
    
    let result = hasher.finalize();
    Ok(format!("{:x}", result))
}

/// Calculate SHA512 hash of a file
pub fn sha512_file(path: &Path) -> Result<String> {
    let file = File::open(path)
        .context(format!("Failed to open file: {:?}", path))?;
    
    let mut reader = BufReader::new(file);
    let mut hasher = Sha512::new();
    let mut buffer = [0u8; 8192];
    
    loop {
        let bytes_read = reader.read(&mut buffer)?;
        if bytes_read == 0 {
            break;
        }
        hasher.update(&buffer[..bytes_read]);
    }
    
    let result = hasher.finalize();
    Ok(format!("{:x}", result))
}

/// Calculate hash of data
pub fn hash_data(data: &[u8], algorithm: HashAlgorithm) -> String {
    match algorithm {
        HashAlgorithm::Sha256 => {
            let mut hasher = Sha256::new();
            hasher.update(data);
            format!("{:x}", hasher.finalize())
        }
        HashAlgorithm::Sha512 => {
            let mut hasher = Sha512::new();
            hasher.update(data);
            format!("{:x}", hasher.finalize())
        }
        HashAlgorithm::Md5 => {
            // MD5 is deprecated but sometimes needed for compatibility
            let digest = md5::compute(data);
            format!("{:x}", digest)
        }
    }
}

/// Verify file checksum
pub fn verify_checksum(path: &Path, expected: &str, algorithm: HashAlgorithm) -> Result<bool> {
    let computed = match algorithm {
        HashAlgorithm::Sha256 => sha256_file(path)?,
        HashAlgorithm::Sha512 => sha512_file(path)?,
        HashAlgorithm::Md5 => {
            let data = std::fs::read(path)?;
            hash_data(&data, HashAlgorithm::Md5)
        }
    };
    
    Ok(computed.to_lowercase() == expected.to_lowercase())
}

/// Parse checksum file (format: "hash  filename")
pub fn parse_checksum_file(content: &str) -> Vec<(String, String)> {
    content.lines()
        .filter_map(|line| {
            let line = line.trim();
            if line.is_empty() || line.starts_with('#') {
                return None;
            }
            
            // Try "hash  filename" format
            if let Some((hash, filename)) = line.split_once("  ") {
                return Some((hash.trim().to_string(), filename.trim().to_string()));
            }
            
            // Try "hash filename" format
            if let Some((hash, filename)) = line.split_once(' ') {
                return Some((hash.trim().to_string(), filename.trim().to_string()));
            }
            
            None
        })
        .collect()
}

/// GPG signature verification (placeholder - requires gpg binary)
pub struct GpgVerifier {
    keyring_path: Option<String>,
}

impl GpgVerifier {
    pub fn new() -> Self {
        Self { keyring_path: None }
    }
    
    pub fn with_keyring(keyring: &str) -> Self {
        Self { keyring_path: Some(keyring.to_string()) }
    }
    
    /// Verify detached signature
    pub fn verify_detached(&self, file: &Path, signature: &Path) -> Result<bool> {
        let mut cmd = std::process::Command::new("gpg");
        cmd.arg("--verify");
        
        if let Some(ref keyring) = self.keyring_path {
            cmd.arg("--keyring").arg(keyring);
        }
        
        cmd.arg(signature).arg(file);
        
        let output = cmd.output()
            .context("Failed to execute gpg")?;
        
        Ok(output.status.success())
    }
    
    /// Import a public key
    pub fn import_key(&self, key_path: &Path) -> Result<()> {
        let mut cmd = std::process::Command::new("gpg");
        cmd.arg("--import");
        
        if let Some(ref keyring) = self.keyring_path {
            cmd.arg("--keyring").arg(keyring);
        }
        
        cmd.arg(key_path);
        
        let output = cmd.output()
            .context("Failed to import key")?;
        
        if !output.status.success() {
            bail!("Failed to import GPG key");
        }
        
        Ok(())
    }
    
    /// Import key from keyserver
    pub fn import_from_keyserver(&self, key_id: &str, keyserver: &str) -> Result<()> {
        let mut cmd = std::process::Command::new("gpg");
        cmd.arg("--keyserver").arg(keyserver);
        cmd.arg("--recv-keys").arg(key_id);
        
        if let Some(ref keyring) = self.keyring_path {
            cmd.arg("--keyring").arg(keyring);
        }
        
        let output = cmd.output()
            .context("Failed to receive key from keyserver")?;
        
        if !output.status.success() {
            bail!("Failed to import key from keyserver");
        }
        
        Ok(())
    }
}

impl Default for GpgVerifier {
    fn default() -> Self {
        Self::new()
    }
}

/// Simple HMAC implementation for package signing
pub fn hmac_sha256(key: &[u8], data: &[u8]) -> String {
    use sha2::Sha256;
    
    const BLOCK_SIZE: usize = 64;
    
    // Prepare key
    let key = if key.len() > BLOCK_SIZE {
        let mut hasher = Sha256::new();
        hasher.update(key);
        hasher.finalize().to_vec()
    } else {
        key.to_vec()
    };
    
    let mut key_padded = vec![0u8; BLOCK_SIZE];
    key_padded[..key.len()].copy_from_slice(&key);
    
    // Inner padding
    let mut inner_key = vec![0x36u8; BLOCK_SIZE];
    for (i, b) in key_padded.iter().enumerate() {
        inner_key[i] ^= b;
    }
    
    // Outer padding
    let mut outer_key = vec![0x5cu8; BLOCK_SIZE];
    for (i, b) in key_padded.iter().enumerate() {
        outer_key[i] ^= b;
    }
    
    // Inner hash
    let mut inner_hasher = Sha256::new();
    inner_hasher.update(&inner_key);
    inner_hasher.update(data);
    let inner_hash = inner_hasher.finalize();
    
    // Outer hash
    let mut outer_hasher = Sha256::new();
    outer_hasher.update(&outer_key);
    outer_hasher.update(&inner_hash);
    let result = outer_hasher.finalize();
    
    format!("{:x}", result)
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_hash_data() {
        let data = b"hello world";
        let hash = hash_data(data, HashAlgorithm::Sha256);
        assert_eq!(
            hash,
            "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
        );
    }
    
    #[test]
    fn test_parse_checksum_file() {
        let content = "abc123  file1.txt\ndef456  file2.txt";
        let checksums = parse_checksum_file(content);
        assert_eq!(checksums.len(), 2);
        assert_eq!(checksums[0], ("abc123".to_string(), "file1.txt".to_string()));
    }
    
    #[test]
    fn test_hmac() {
        let key = b"secret";
        let data = b"message";
        let hmac = hmac_sha256(key, data);
        assert!(!hmac.is_empty());
    }
}
