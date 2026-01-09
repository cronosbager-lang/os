use anyhow::{Result, Context, bail};
use std::path::Path;
use std::io::Write;
use std::process::Command;

pub fn sync_database(repo_url: &str, repo_name: &str) -> Result<()> {
    let db_url = format!("{}/{}.db", repo_url, repo_name);
    let db_path = format!("/var/lib/mix-pkg/db/{}.db", repo_name);
    
    std::fs::create_dir_all("/var/lib/mix-pkg/db")?;
    
    // Download database file
    download_file(&db_url, Path::new(&db_path), |_| {})?;
    
    Ok(())
}

pub fn download_file<F>(url: &str, dest: &Path, _progress: F) -> Result<()>
where
    F: Fn(u64),
{
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
    
    // Create parent directory if needed
    if let Some(parent) = dest.parent() {
        std::fs::create_dir_all(parent)?;
    }
    
    let mut file = std::fs::File::create(dest)
        .context(format!("Failed to create file {:?}", dest))?;
    
    let bytes = output.stdout;
    file.write_all(&bytes)?;
    
    Ok(())
}

