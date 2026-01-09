// Download utilities for mix-pkg
use anyhow::{Result, Context, bail};
use std::path::Path;
use std::fs::{self, File};
use std::io::Write;
use std::process::Command;

/// Download a file from URL to destination
pub fn download_file(url: &str, dest: &Path) -> Result<()> {
    download_file_with_progress(url, dest, |_| {})
}

/// Download a file with progress callback
pub fn download_file_with_progress<F>(url: &str, dest: &Path, mut progress: F) -> Result<()>
where
    F: FnMut(u64),
{
    // Create parent directory if needed
    if let Some(parent) = dest.parent() {
        fs::create_dir_all(parent)?;
    }
    
    // Use system curl command for downloads (no Rust HTTP library dependencies)
    let output = Command::new("curl")
        .arg("-fsSL")
        .arg("--max-time")
        .arg("300")
        .arg(url)
        .output()
        .context(format!("Failed to download from {}", url))?;
    
    if !output.status.success() {
        bail!("curl download failed: {}", String::from_utf8_lossy(&output.stderr));
    }
    
    let mut file = File::create(dest)
        .context(format!("Failed to create file: {:?}", dest))?;
    
    let bytes = output.stdout;
    file.write_all(&bytes)
        .context("Failed to write to file")?;
    
    file.flush()?;
    progress(bytes.len() as u64);
    
    Ok(())
}

/// Download with retry
pub fn download_with_retry(url: &str, dest: &Path, retries: u32) -> Result<()> {
    let mut last_error = None;
    
    for attempt in 0..retries {
        match download_file(url, dest) {
            Ok(()) => return Ok(()),
            Err(e) => {
                last_error = Some(e);
                if attempt < retries - 1 {
                    // Wait before retry with exponential backoff
                    let wait_time = std::time::Duration::from_secs(2u64.pow(attempt));
                    std::thread::sleep(wait_time);
                }
            }
        }
    }
    
    Err(last_error.unwrap())
}

/// Check if URL is reachable
pub fn check_url(url: &str) -> Result<bool> {
    let output = Command::new("curl")
        .arg("-fsS")
        .arg("--max-time")
        .arg("10")
        .arg("-I")
        .arg(url)
        .output()
        .context(format!("Failed to check URL: {}", url))?;
    
    Ok(output.status.success())
}

/// Get file size from URL without downloading
pub fn get_remote_size(url: &str) -> Result<Option<u64>> {
    let output = Command::new("curl")
        .arg("-fsS")
        .arg("--max-time")
        .arg("10")
        .arg("-I")
        .arg(url)
        .output()
        .context("Failed to get file info")?;
    
    if !output.status.success() {
        return Ok(None);
    }
    
    let headers = String::from_utf8_lossy(&output.stdout);
    for line in headers.lines() {
        if line.to_lowercase().starts_with("content-length:") {
            if let Ok(size) = line.split(':').nth(1)
                .unwrap_or("0")
                .trim()
                .parse::<u64>()
            {
                return Ok(Some(size));
            }
        }
    }
    
    Ok(None)
}

/// Resume download from partial file
pub fn resume_download<F>(url: &str, dest: &Path, _progress: F) -> Result<()>
where
    F: FnMut(u64, u64), // (downloaded, total)
{
    let existing_size = if dest.exists() {
        fs::metadata(dest)?.len()
    } else {
        0
    };
    
    // Get total size
    let total_size = get_remote_size(url)?.unwrap_or(0);
    
    // Check if already complete
    if existing_size >= total_size && total_size > 0 {
        return Ok(());
    }
    
    // Create parent directory if needed
    if let Some(parent) = dest.parent() {
        fs::create_dir_all(parent)?;
    }
    
    // Download with range header if resumable
    let range_arg = if existing_size > 0 {
        format!("{}-", existing_size)
    } else {
        "0-".to_string()
    };
    
    let output = Command::new("curl")
        .arg("-fsSL")
        .arg("--max-time").arg("300")
        .arg("-r").arg(&range_arg)
        .arg("-o").arg(dest)
        .arg(url)
        .output()
        .context("Failed to resume download")?;
    
    if !output.status.success() {
        bail!("curl download failed: {}", String::from_utf8_lossy(&output.stderr));
    }
    
    Ok(())
}

/// Mirror selection - find fastest mirror
pub fn select_fastest_mirror(mirrors: &[String]) -> Option<String> {
    let mut fastest: Option<(String, std::time::Duration)> = None;
    
    for mirror in mirrors {
        let start = std::time::Instant::now();
        
        let output = Command::new("curl")
            .arg("-fsS")
            .arg("--max-time").arg("5")
            .arg("-I")
            .arg(mirror)
            .output();
        
        if let Ok(output) = output {
            if output.status.success() {
                let elapsed = start.elapsed();
                
                if fastest.is_none() || elapsed < fastest.as_ref().unwrap().1 {
                    fastest = Some((mirror.clone(), elapsed));
                }
            }
        }
    }
    
    fastest.map(|(url, _)| url)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::env::temp_dir;
    
    #[test]
    fn test_check_url() {
        // This test requires network access
        let result = check_url("https://example.com");
        assert!(result.is_ok());
    }
}
