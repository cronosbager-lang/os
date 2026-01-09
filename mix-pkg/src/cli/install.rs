use anyhow::{Result, Context};
use colored::Colorize;
use indicatif::{ProgressBar, ProgressStyle};
use std::io::{self, Write};

use crate::core::{database, package, repository};
use crate::config::Config;

pub fn run(packages: &[String], force: bool, yes: bool, verbose: bool) -> Result<()> {
    let config = Config::load()?;
    
    println!("{} Resolving packages...", "→".cyan());
    
    // Load package database
    let db = database::Database::load(&config)?;
    
    // Resolve dependencies
    let mut to_install: Vec<package::Package> = Vec::new();
    let mut not_found: Vec<String> = Vec::new();
    
    for pkg_name in packages {
        if let Some(pkg) = db.find_package(pkg_name) {
            // Check if already installed
            if !force && db.is_installed(pkg_name) {
                println!("  {} {} is already installed", "→".cyan(), pkg_name);
                continue;
            }
            
            // Add package and its dependencies
            let deps = db.resolve_dependencies(&pkg)?;
            for dep in deps {
                if !to_install.iter().any(|p| p.name == dep.name) {
                    to_install.push(dep);
                }
            }
            to_install.push(pkg);
        } else {
            not_found.push(pkg_name.clone());
        }
    }
    
    if !not_found.is_empty() {
        println!("{} Packages not found: {}", "✗".red(), not_found.join(", "));
        return Ok(());
    }
    
    if to_install.is_empty() {
        println!("{} Nothing to install", "✓".green());
        return Ok(());
    }
    
    // Show what will be installed
    println!();
    println!("Packages to install ({}):", to_install.len());
    for pkg in &to_install {
        println!("  {} {}-{}", "•".cyan(), pkg.name, pkg.version);
    }
    
    // Calculate total size
    let total_size: u64 = to_install.iter().map(|p| p.size).sum();
    println!();
    println!("Total download size: {}", format_size(total_size));
    println!();
    
    // Confirm
    if !yes {
        print!("Proceed with installation? [Y/n] ");
        io::stdout().flush()?;
        
        let mut input = String::new();
        io::stdin().read_line(&mut input)?;
        let input = input.trim().to_lowercase();
        
        if !input.is_empty() && input != "y" && input != "yes" {
            println!("Installation cancelled.");
            return Ok(());
        }
    }
    
    // Download and install packages
    for pkg in &to_install {
        install_package(&config, pkg, verbose)?;
    }
    
    println!();
    println!("{} {} package(s) installed successfully", "✓".green(), to_install.len());
    
    Ok(())
}

fn install_package(config: &Config, pkg: &package::Package, verbose: bool) -> Result<()> {
    println!();
    println!("{} Installing {}...", "→".cyan(), pkg.name);
    
    // Download package
    let pb = ProgressBar::new(pkg.size);
    pb.set_style(ProgressStyle::default_bar()
        .template("  [{bar:40.cyan/blue}] {bytes}/{total_bytes} ({eta})")
        .progress_chars("█▓░"));
    
    let cache_dir = dirs::cache_dir()
        .unwrap_or_else(|| std::path::PathBuf::from("/var/cache"))
        .join("mix-pkg");
    std::fs::create_dir_all(&cache_dir)?;
    
    let pkg_file = cache_dir.join(format!("{}-{}.pkg.tar.zst", pkg.name, pkg.version));
    
    // Download if not cached
    if !pkg_file.exists() {
        let url = format!("{}/pool/{}/{}-{}.pkg.tar.zst", 
            config.repositories.first().map(|r| r.url.as_str()).unwrap_or(""),
            pkg.repository,
            pkg.name,
            pkg.version
        );
        
        if verbose {
            println!("  Downloading from: {}", url);
        }
        
        repository::download_file(&url, &pkg_file, |downloaded| {
            pb.set_position(downloaded);
        })?;
    }
    
    pb.finish_and_clear();
    
    // Verify checksum
    if verbose {
        println!("  Verifying checksum...");
    }
    package::verify_checksum(&pkg_file, &pkg.checksum)?;
    
    // Extract package
    if verbose {
        println!("  Extracting package...");
    }
    package::extract_package(&pkg_file, "/")?;
    
    // Run post-install script if exists
    let post_install = format!("/var/lib/mix-pkg/scripts/{}.install", pkg.name);
    if std::path::Path::new(&post_install).exists() {
        if verbose {
            println!("  Running post-install script...");
        }
        std::process::Command::new("sh")
            .arg(&post_install)
            .arg("post_install")
            .status()
            .context("Failed to run post-install script")?;
    }
    
    // Record installation
    database::record_installation(pkg)?;
    
    println!("  {} {} installed", "✓".green(), pkg.name);
    
    Ok(())
}

fn format_size(bytes: u64) -> String {
    const KB: u64 = 1024;
    const MB: u64 = KB * 1024;
    const GB: u64 = MB * 1024;
    
    if bytes >= GB {
        format!("{:.2} GB", bytes as f64 / GB as f64)
    } else if bytes >= MB {
        format!("{:.2} MB", bytes as f64 / MB as f64)
    } else if bytes >= KB {
        format!("{:.2} KB", bytes as f64 / KB as f64)
    } else {
        format!("{} B", bytes)
    }
}
