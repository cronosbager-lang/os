use anyhow::Result;
use colored::Colorize;

use crate::core::database;
use crate::config::Config;

pub fn run_search(query: &str) -> Result<()> {
    let config = Config::load()?;
    let db = database::Database::load(&config)?;
    
    let results = db.search(query);
    
    if results.is_empty() {
        println!("No packages found matching '{}'", query);
        return Ok(());
    }
    
    println!("Search results for '{}':", query);
    println!();
    
    for pkg in results {
        let installed = if db.is_installed(&pkg.name) {
            "[installed]".green().to_string()
        } else {
            String::new()
        };
        
        println!("{}/{} {} {}", 
            pkg.repository.cyan(),
            pkg.name.bold(),
            pkg.version,
            installed
        );
        println!("    {}", pkg.description);
    }
    
    Ok(())
}

pub fn run_info(package: &str) -> Result<()> {
    let config = Config::load()?;
    let db = database::Database::load(&config)?;
    
    if let Some(pkg) = db.find_package(package) {
        println!("{}", "Package Information".cyan().bold());
        println!("{}", "─".repeat(40));
        println!("{:15} {}", "Name:".bold(), pkg.name);
        println!("{:15} {}", "Version:".bold(), pkg.version);
        println!("{:15} {}", "Description:".bold(), pkg.description);
        println!("{:15} {}", "Repository:".bold(), pkg.repository);
        println!("{:15} {}", "Size:".bold(), format_size(pkg.size));
        
        if !pkg.dependencies.is_empty() {
            println!("{:15} {}", "Dependencies:".bold(), pkg.dependencies.join(", "));
        }
        
        if db.is_installed(package) {
            println!("{:15} {}", "Status:".bold(), "Installed".green());
        } else {
            println!("{:15} {}", "Status:".bold(), "Not installed".yellow());
        }
    } else {
        println!("Package '{}' not found", package);
    }
    
    Ok(())
}

pub fn run_list() -> Result<()> {
    let config = Config::load()?;
    let db = database::Database::load(&config)?;
    
    let installed = db.list_installed();
    
    if installed.is_empty() {
        println!("No packages installed");
        return Ok(());
    }
    
    println!("{}", "Installed Packages".cyan().bold());
    println!("{}", "─".repeat(60));
    
    for (name, version) in installed {
        println!("{} {}", name.bold(), version);
    }
    
    Ok(())
}

fn format_size(bytes: u64) -> String {
    const KB: u64 = 1024;
    const MB: u64 = KB * 1024;
    
    if bytes >= MB {
        format!("{:.2} MB", bytes as f64 / MB as f64)
    } else if bytes >= KB {
        format!("{:.2} KB", bytes as f64 / KB as f64)
    } else {
        format!("{} B", bytes)
    }
}
