use anyhow::Result;
use colored::Colorize;
use std::io::{self, Write};

use crate::core::{database, repository};
use crate::config::Config;

pub fn run_update(verbose: bool) -> Result<()> {
    let config = Config::load()?;
    
    println!("{} Updating package database...", "→".cyan());
    
    for repo in &config.repositories {
        println!("  {} Syncing {}...", "→".cyan(), repo.name);
        
        if verbose {
            println!("    URL: {}", repo.url);
        }
        
        match repository::sync_database(&repo.url, &repo.name) {
            Ok(_) => println!("    {} {} synced", "✓".green(), repo.name),
            Err(e) => println!("    {} Failed to sync {}: {}", "✗".red(), repo.name, e),
        }
    }
    
    println!();
    println!("{} Package database updated", "✓".green());
    
    Ok(())
}

pub fn run_upgrade(yes: bool, verbose: bool) -> Result<()> {
    let config = Config::load()?;
    
    // First update database
    run_update(verbose)?;
    
    println!();
    println!("{} Checking for upgrades...", "→".cyan());
    
    let db = database::Database::load(&config)?;
    let upgrades = db.get_upgradable();
    
    if upgrades.is_empty() {
        println!("{} System is up to date", "✓".green());
        return Ok(());
    }
    
    println!();
    println!("Packages to upgrade ({}):", upgrades.len());
    for (name, old_ver, new_ver) in &upgrades {
        println!("  {} {} -> {}", name.bold(), old_ver.yellow(), new_ver.green());
    }
    println!();
    
    // Confirm
    if !yes {
        print!("Proceed with upgrade? [Y/n] ");
        io::stdout().flush()?;
        
        let mut input = String::new();
        io::stdin().read_line(&mut input)?;
        let input = input.trim().to_lowercase();
        
        if !input.is_empty() && input != "y" && input != "yes" {
            println!("Upgrade cancelled.");
            return Ok(());
        }
    }
    
    // Upgrade packages
    let packages: Vec<String> = upgrades.iter().map(|(name, _, _)| name.clone()).collect();
    super::install::run(&packages, true, true, verbose)?;
    
    Ok(())
}
