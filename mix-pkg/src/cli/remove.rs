use anyhow::Result;
use colored::Colorize;
use std::io::{self, Write};

use crate::core::database;
use crate::config::Config;

pub fn run(packages: &[String], yes: bool, _verbose: bool) -> Result<()> {
    let config = Config::load()?;
    let db = database::Database::load(&config)?;
    
    println!("{} Checking packages...", "→".cyan());
    
    let mut to_remove: Vec<String> = Vec::new();
    let mut not_installed: Vec<String> = Vec::new();
    
    for pkg_name in packages {
        if db.is_installed(pkg_name) {
            to_remove.push(pkg_name.clone());
        } else {
            not_installed.push(pkg_name.clone());
        }
    }
    
    if !not_installed.is_empty() {
        println!("{} Not installed: {}", "→".yellow(), not_installed.join(", "));
    }
    
    if to_remove.is_empty() {
        println!("{} Nothing to remove", "✓".green());
        return Ok(());
    }
    
    // Check for dependents
    let mut dependents: Vec<String> = Vec::new();
    for pkg_name in &to_remove {
        let deps = db.get_dependents(pkg_name);
        for dep in deps {
            if !to_remove.contains(&dep) && !dependents.contains(&dep) {
                dependents.push(dep);
            }
        }
    }
    
    if !dependents.is_empty() {
        println!();
        println!("{} The following packages depend on packages being removed:", "!".yellow());
        for dep in &dependents {
            println!("  {} {}", "•".yellow(), dep);
        }
    }
    
    // Show what will be removed
    println!();
    println!("Packages to remove ({}):", to_remove.len());
    for pkg in &to_remove {
        println!("  {} {}", "•".red(), pkg);
    }
    println!();
    
    // Confirm
    if !yes {
        print!("Proceed with removal? [y/N] ");
        io::stdout().flush()?;
        
        let mut input = String::new();
        io::stdin().read_line(&mut input)?;
        let input = input.trim().to_lowercase();
        
        if input != "y" && input != "yes" {
            println!("Removal cancelled.");
            return Ok(());
        }
    }
    
    // Remove packages
    for pkg_name in &to_remove {
        remove_package(pkg_name)?;
    }
    
    println!();
    println!("{} {} package(s) removed", "✓".green(), to_remove.len());
    
    Ok(())
}

fn remove_package(pkg_name: &str) -> Result<()> {
    println!("{} Removing {}...", "→".cyan(), pkg_name);
    
    // Run pre-remove script if exists
    let pre_remove = format!("/var/lib/mix-pkg/scripts/{}.install", pkg_name);
    if std::path::Path::new(&pre_remove).exists() {
        std::process::Command::new("sh")
            .arg(&pre_remove)
            .arg("pre_remove")
            .status()?;
    }
    
    // Get file list
    let files_list = format!("/var/lib/mix-pkg/installed/{}/files", pkg_name);
    if let Ok(content) = std::fs::read_to_string(&files_list) {
        // Remove files in reverse order (deepest first)
        let mut files: Vec<&str> = content.lines().collect();
        files.reverse();
        
        for file in files {
            let path = std::path::Path::new(file);
            if path.is_file() {
                let _ = std::fs::remove_file(path);
            } else if path.is_dir() {
                // Only remove if empty
                let _ = std::fs::remove_dir(path);
            }
        }
    }
    
    // Remove package record
    let pkg_dir = format!("/var/lib/mix-pkg/installed/{}", pkg_name);
    let _ = std::fs::remove_dir_all(&pkg_dir);
    
    println!("  {} {} removed", "✓".green(), pkg_name);
    
    Ok(())
}
