use anyhow::{Result, Context};
use std::path::Path;
use std::io::Write;

pub fn sync_database(repo_url: &str, repo_name: &str) -> Result<()> {
    let db_url = format!("{}/{}.db", repo_url, repo_name);
    let db_path = format!("/var/lib/mix-pkg/db/{}.db", repo_name);
    
    std::fs::create_dir_all("/var/lib/mix-pkg/db")?;
    
    // Download database file
    download_file(&db_url, Path::new(&db_path), |_| {})?;
    
    Ok(())
}

pub fn download_file<F>(url: &str, dest: &Path, progress: F) -> Result<()>
where
    F: Fn(u64),
{
    // For now, use a simple blocking download
    // In production, this would use async reqwest with proper progress
    
    let response = reqwest::blocking::get(url)
        .context(format!("Failed to download from {}", url))?;
    
    if !response.status().is_success() {
        anyhow::bail!("HTTP error: {}", response.status());
    }
    
    let total_size = response.content_length().unwrap_or(0);
    let bytes = response.bytes()?;
    
    // Create parent directory if needed
    if let Some(parent) = dest.parent() {
        std::fs::create_dir_all(parent)?;
    }
    
    let mut file = std::fs::File::create(dest)
        .context(format!("Failed to create file {:?}", dest))?;
    
    file.write_all(&bytes)?;
    progress(total_size);
    
    Ok(())
}
