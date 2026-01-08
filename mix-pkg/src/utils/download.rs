// Download utilities for mix-pkg
use anyhow::{Result, Context, bail};
use std::path::Path;
use std::fs::{self, File};
use std::io::{Write, Read};

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
    
    // Use reqwest for HTTP downloads
    let response = reqwest::blocking::Client::new()
        .get(url)
        .timeout(std::time::Duration::from_secs(300))
        .send()
        .context(format!("Failed to connect to {}", url))?;
    
    if !response.status().is_success() {
        bail!("HTTP error {}: {}", response.status(), url);
    }
    
    let total_size = response.content_length().unwrap_or(0);
    let mut downloaded: u64 = 0;
    
    let mut file = File::create(dest)
        .context(format!("Failed to create file: {:?}", dest))?;
    
    let mut reader = response;
    let mut buffer = [0u8; 8192];
    
    loop {
        let bytes_read = reader.read(&mut buffer)
            .context("Failed to read from response")?;
        
        if bytes_read == 0 {
            break;
        }
        
        file.write_all(&buffer[..bytes_read])
            .context("Failed to write to file")?;
        
        downloaded += bytes_read as u64;
        progress(downloaded);
    }
    
    file.flush()?;
    
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
    let response = reqwest::blocking::Client::new()
        .head(url)
        .timeout(std::time::Duration::from_secs(10))
        .send();
    
    match response {
        Ok(r) => Ok(r.status().is_success()),
        Err(_) => Ok(false),
    }
}

/// Get file size from URL without downloading
pub fn get_remote_size(url: &str) -> Result<Option<u64>> {
    let response = reqwest::blocking::Client::new()
        .head(url)
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .context("Failed to get file info")?;
    
    Ok(response.content_length())
}

/// Resume download from partial file
pub fn resume_download<F>(url: &str, dest: &Path, mut progress: F) -> Result<()>
where
    F: FnMut(u64, u64), // (downloaded, total)
{
    let existing_size = if dest.exists() {
        fs::metadata(dest)?.len()
    } else {
        0
    };
    
    let client = reqwest::blocking::Client::new();
    
    // Get total size
    let head_response = client.head(url)
        .timeout(std::time::Duration::from_secs(10))
        .send()?;
    
    let total_size = head_response.content_length().unwrap_or(0);
    
    // Check if already complete
    if existing_size >= total_size && total_size > 0 {
        progress(total_size, total_size);
        return Ok(());
    }
    
    // Request with Range header
    let response = client.get(url)
        .header("Range", format!("bytes={}-", existing_size))
        .timeout(std::time::Duration::from_secs(300))
        .send()
        .context("Failed to resume download")?;
    
    // Check if server supports range requests
    let supports_resume = response.status() == reqwest::StatusCode::PARTIAL_CONTENT;
    
    let mut file = if supports_resume && existing_size > 0 {
        fs::OpenOptions::new()
            .append(true)
            .open(dest)?
    } else {
        File::create(dest)?
    };
    
    let mut downloaded = if supports_resume { existing_size } else { 0 };
    let mut reader = response;
    let mut buffer = [0u8; 8192];
    
    loop {
        let bytes_read = reader.read(&mut buffer)?;
        if bytes_read == 0 {
            break;
        }
        
        file.write_all(&buffer[..bytes_read])?;
        downloaded += bytes_read as u64;
        progress(downloaded, total_size);
    }
    
    file.flush()?;
    Ok(())
}

/// Mirror selection - find fastest mirror
pub fn select_fastest_mirror(mirrors: &[String]) -> Option<String> {
    let mut fastest: Option<(String, std::time::Duration)> = None;
    
    for mirror in mirrors {
        let start = std::time::Instant::now();
        
        if let Ok(response) = reqwest::blocking::Client::new()
            .head(mirror)
            .timeout(std::time::Duration::from_secs(5))
            .send()
        {
            if response.status().is_success() {
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
