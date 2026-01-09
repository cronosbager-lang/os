// Compression utilities for mix-pkg
use anyhow::{Result, Context, bail};
use std::path::Path;
use std::fs::{self, File};
use std::io::{Read, Write, BufReader, BufWriter};

/// Supported compression formats
#[derive(Debug, Clone, Copy, PartialEq)]
pub enum CompressionFormat {
    None,
    Gzip,
    Xz,
    Zstd,
    Bzip2,
}

impl CompressionFormat {
    /// Detect format from file extension
    pub fn from_extension(path: &Path) -> Self {
        let ext = path.extension()
            .and_then(|e| e.to_str())
            .unwrap_or("");
        
        match ext {
            "gz" | "gzip" => Self::Gzip,
            "xz" => Self::Xz,
            "zst" | "zstd" => Self::Zstd,
            "bz2" | "bzip2" => Self::Bzip2,
            _ => Self::None,
        }
    }
    
    /// Detect format from magic bytes
    pub fn from_magic(data: &[u8]) -> Self {
        if data.len() < 4 {
            return Self::None;
        }
        
        // Gzip: 1f 8b
        if data[0] == 0x1f && data[1] == 0x8b {
            return Self::Gzip;
        }
        
        // XZ: fd 37 7a 58 5a 00
        if data.len() >= 6 && data[..6] == [0xfd, 0x37, 0x7a, 0x58, 0x5a, 0x00] {
            return Self::Xz;
        }
        
        // Zstd: 28 b5 2f fd
        if data[..4] == [0x28, 0xb5, 0x2f, 0xfd] {
            return Self::Zstd;
        }
        
        // Bzip2: 42 5a 68
        if data.len() >= 3 && data[..3] == [0x42, 0x5a, 0x68] {
            return Self::Bzip2;
        }
        
        Self::None
    }
}

/// Extract a tar.zst archive
pub fn extract_tar_zst(archive: &Path, dest: &Path) -> Result<Vec<String>> {
    let file = File::open(archive)
        .context(format!("Failed to open archive: {:?}", archive))?;
    
    let decoder = zstd::stream::Decoder::new(file)
        .context("Failed to create zstd decoder")?;
    
    extract_tar(decoder, dest)
}

/// Extract a tar.gz archive
pub fn extract_tar_gz(archive: &Path, dest: &Path) -> Result<Vec<String>> {
    let file = File::open(archive)
        .context(format!("Failed to open archive: {:?}", archive))?;
    
    let decoder = flate2::read::GzDecoder::new(file);
    
    extract_tar(decoder, dest)
}

/// Extract a tar.xz archive
pub fn extract_tar_xz(archive: &Path, dest: &Path) -> Result<Vec<String>> {
    let file = File::open(archive)
        .context(format!("Failed to open archive: {:?}", archive))?;
    
    let decoder = xz2::read::XzDecoder::new(file);
    
    extract_tar(decoder, dest)
}

/// Extract tar archive from reader
fn extract_tar<R: Read>(reader: R, dest: &Path) -> Result<Vec<String>> {
    let mut archive = tar::Archive::new(reader);
    archive.set_preserve_permissions(true);
    archive.set_preserve_mtime(true);
    archive.set_overwrite(true);
    
    let mut extracted_files = Vec::new();
    
    for entry in archive.entries()? {
        let mut entry = entry?;
        let path = entry.path()?.to_path_buf();
        
        // Skip metadata files
        let path_str = path.to_string_lossy();
        if path_str.starts_with('.') && !path_str.starts_with("./") {
            continue;
        }
        
        let dest_path = dest.join(&path);
        extracted_files.push(dest_path.to_string_lossy().to_string());
        
        // Create parent directories
        if let Some(parent) = dest_path.parent() {
            fs::create_dir_all(parent)?;
        }
        
        entry.unpack(&dest_path)
            .context(format!("Failed to extract: {:?}", path))?;
    }
    
    Ok(extracted_files)
}

/// Create a tar.zst archive
pub fn create_tar_zst(source: &Path, archive: &Path, level: i32) -> Result<()> {
    let file = File::create(archive)
        .context(format!("Failed to create archive: {:?}", archive))?;
    
    let encoder = zstd::stream::Encoder::new(file, level)?
        .auto_finish();
    
    create_tar(source, encoder)
}

/// Create a tar.gz archive
pub fn create_tar_gz(source: &Path, archive: &Path, level: u32) -> Result<()> {
    let file = File::create(archive)
        .context(format!("Failed to create archive: {:?}", archive))?;
    
    let encoder = flate2::write::GzEncoder::new(file, flate2::Compression::new(level));
    
    create_tar(source, encoder)
}

/// Create tar archive to writer
fn create_tar<W: Write>(source: &Path, writer: W) -> Result<()> {
    let mut builder = tar::Builder::new(writer);
    
    if source.is_dir() {
        builder.append_dir_all(".", source)?;
    } else {
        let name = source.file_name()
            .ok_or_else(|| anyhow::anyhow!("Invalid source path"))?;
        builder.append_path_with_name(source, name)?;
    }
    
    builder.finish()?;
    Ok(())
}

/// Decompress a single file
pub fn decompress_file(input: &Path, output: &Path) -> Result<()> {
    let format = CompressionFormat::from_extension(input);
    
    let input_file = File::open(input)?;
    let mut output_file = File::create(output)?;
    
    match format {
        CompressionFormat::Gzip => {
            let mut decoder = flate2::read::GzDecoder::new(input_file);
            std::io::copy(&mut decoder, &mut output_file)?;
        }
        CompressionFormat::Xz => {
            let mut decoder = xz2::read::XzDecoder::new(input_file);
            std::io::copy(&mut decoder, &mut output_file)?;
        }
        CompressionFormat::Zstd => {
            let mut decoder = zstd::stream::Decoder::new(input_file)?;
            std::io::copy(&mut decoder, &mut output_file)?;
        }
        CompressionFormat::None | CompressionFormat::Bzip2 => {
            bail!("Unsupported compression format");
        }
    }
    
    Ok(())
}

/// Compress a single file
pub fn compress_file(input: &Path, output: &Path, format: CompressionFormat, level: i32) -> Result<()> {
    let mut input_file = File::open(input)?;
    let output_file = File::create(output)?;
    
    match format {
        CompressionFormat::Gzip => {
            let mut encoder = flate2::write::GzEncoder::new(
                output_file, 
                flate2::Compression::new(level as u32)
            );
            std::io::copy(&mut input_file, &mut encoder)?;
            encoder.finish()?;
        }
        CompressionFormat::Xz => {
            let mut encoder = xz2::write::XzEncoder::new(output_file, level as u32);
            std::io::copy(&mut input_file, &mut encoder)?;
            encoder.finish()?;
        }
        CompressionFormat::Zstd => {
            let mut encoder = zstd::stream::Encoder::new(output_file, level)?;
            std::io::copy(&mut input_file, &mut encoder)?;
            encoder.finish()?;
        }
        CompressionFormat::None | CompressionFormat::Bzip2 => {
            bail!("Unsupported compression format");
        }
    }
    
    Ok(())
}

/// Get uncompressed size estimate
pub fn estimate_uncompressed_size(path: &Path) -> Result<u64> {
    let file = File::open(path)?;
    let format = CompressionFormat::from_extension(path);
    
    match format {
        CompressionFormat::Gzip => {
            // Gzip stores original size in last 4 bytes
            let metadata = file.metadata()?;
            let file_size = metadata.len();
            
            if file_size < 4 {
                return Ok(0);
            }
            
            let mut file = file;
            use std::io::Seek;
            file.seek(std::io::SeekFrom::End(-4))?;
            
            let mut buf = [0u8; 4];
            file.read_exact(&mut buf)?;
            
            Ok(u32::from_le_bytes(buf) as u64)
        }
        _ => {
            // For other formats, estimate based on compression ratio
            let metadata = file.metadata()?;
            Ok(metadata.len() * 3) // Rough estimate
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_compression_format_detection() {
        assert_eq!(
            CompressionFormat::from_extension(Path::new("file.tar.gz")),
            CompressionFormat::Gzip
        );
        assert_eq!(
            CompressionFormat::from_extension(Path::new("file.tar.zst")),
            CompressionFormat::Zstd
        );
        assert_eq!(
            CompressionFormat::from_extension(Path::new("file.tar.xz")),
            CompressionFormat::Xz
        );
    }
    
    #[test]
    fn test_magic_detection() {
        assert_eq!(
            CompressionFormat::from_magic(&[0x1f, 0x8b, 0x08, 0x00]),
            CompressionFormat::Gzip
        );
        assert_eq!(
            CompressionFormat::from_magic(&[0x28, 0xb5, 0x2f, 0xfd]),
            CompressionFormat::Zstd
        );
    }
}
