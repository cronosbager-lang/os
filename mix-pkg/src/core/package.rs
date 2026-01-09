use anyhow::{Result, Context, bail};
use serde::{Deserialize, Serialize};
use sha2::{Sha256, Digest};
use std::path::Path;
use std::io::Read;

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct Package {
    pub name: String,
    pub version: String,
    pub description: String,
    #[serde(default)]
    pub repository: String,
    #[serde(default)]
    pub size: u64,
    #[serde(default)]
    pub installed_size: u64,
    #[serde(default)]
    pub checksum: String,
    #[serde(default)]
    pub dependencies: Vec<String>,
    #[serde(default)]
    pub provides: Vec<String>,
    #[serde(default)]
    pub conflicts: Vec<String>,
    #[serde(default)]
    pub url: String,
    #[serde(default)]
    pub license: String,
    #[serde(default)]
    pub maintainer: String,
    #[serde(default)]
    pub build_date: u64,
    #[serde(default)]
    pub install_date: u64,
}

impl Package {
    pub fn from_pkginfo(content: &str, repo: &str) -> Result<Self> {
        let mut pkg = Package {
            name: String::new(),
            version: String::new(),
            description: String::new(),
            repository: repo.to_string(),
            size: 0,
            installed_size: 0,
            checksum: String::new(),
            dependencies: Vec::new(),
            provides: Vec::new(),
            conflicts: Vec::new(),
            url: String::new(),
            license: String::new(),
            maintainer: String::new(),
            build_date: 0,
            install_date: 0,
        };
        
        for line in content.lines() {
            let line = line.trim();
            if line.is_empty() || line.starts_with('#') {
                continue;
            }
            
            if let Some((key, value)) = line.split_once('=') {
                let key = key.trim();
                let value = value.trim();
                
                match key {
                    "pkgname" => pkg.name = value.to_string(),
                    "pkgver" => pkg.version = value.to_string(),
                    "pkgdesc" => pkg.description = value.to_string(),
                    "size" => pkg.size = value.parse().unwrap_or(0),
                    "isize" => pkg.installed_size = value.parse().unwrap_or(0),
                    "sha256sum" => pkg.checksum = value.to_string(),
                    "depend" => pkg.dependencies.push(value.to_string()),
                    "provides" => pkg.provides.push(value.to_string()),
                    "conflict" => pkg.conflicts.push(value.to_string()),
                    _ => {}
                }
            }
        }
        
        Ok(pkg)
    }
}

pub fn verify_checksum(file_path: &Path, expected: &str) -> Result<()> {
    let mut file = std::fs::File::open(file_path)
        .context("Failed to open file for checksum verification")?;
    
    let mut hasher = Sha256::new();
    let mut buffer = [0u8; 8192];
    
    loop {
        let bytes_read = file.read(&mut buffer)?;
        if bytes_read == 0 {
            break;
        }
        hasher.update(&buffer[..bytes_read]);
    }
    
    let result = hasher.finalize();
    let computed = format!("{:x}", result);
    
    if computed != expected {
        bail!("Checksum mismatch: expected {}, got {}", expected, computed);
    }
    
    Ok(())
}

pub fn extract_package(pkg_path: &Path, dest: &str) -> Result<()> {
    let file = std::fs::File::open(pkg_path)
        .context("Failed to open package file")?;
    
    // Decompress zstd
    let decoder = zstd::stream::Decoder::new(file)
        .context("Failed to create zstd decoder")?;
    
    // Extract tar
    let mut archive = tar::Archive::new(decoder);
    archive.set_preserve_permissions(true);
    archive.set_preserve_mtime(true);
    
    // Track extracted files
    let mut files = Vec::new();
    
    for entry in archive.entries()? {
        let mut entry = entry?;
        let path = entry.path()?.to_path_buf();
        
        // Skip .PKGINFO and other metadata
        if path.to_string_lossy().starts_with('.') {
            continue;
        }
        
        let dest_path = Path::new(dest).join(&path);
        files.push(dest_path.to_string_lossy().to_string());
        
        entry.unpack(&dest_path)
            .context(format!("Failed to extract {:?}", path))?;
    }
    
    // Save file list for later removal
    let pkg_name = pkg_path.file_stem()
        .and_then(|s| s.to_str())
        .and_then(|s| s.split('-').next())
        .unwrap_or("unknown");
    
    let files_dir = format!("/var/lib/mix-pkg/installed/{}", pkg_name);
    std::fs::create_dir_all(&files_dir)?;
    std::fs::write(format!("{}/files", files_dir), files.join("\n"))?;
    
    Ok(())
}
