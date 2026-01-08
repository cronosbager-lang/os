use anyhow::Result;
use colored::Colorize;

use crate::cli::RepoCommands;
use crate::config::Config;

pub fn run(command: RepoCommands) -> Result<()> {
    match command {
        RepoCommands::List => list_repos(),
        RepoCommands::Add { url, name } => add_repo(&url, name),
        RepoCommands::Remove { name } => remove_repo(&name),
    }
}

fn list_repos() -> Result<()> {
    let config = Config::load()?;
    
    println!("{}", "Configured Repositories".cyan().bold());
    println!("{}", "─".repeat(60));
    
    for repo in &config.repositories {
        let status = if repo.enabled { "●".green() } else { "○".yellow() };
        println!("{} {} ({})", status, repo.name.bold(), repo.url);
    }
    
    Ok(())
}

fn add_repo(url: &str, name: Option<String>) -> Result<()> {
    let mut config = Config::load()?;
    
    let repo_name = name.unwrap_or_else(|| {
        // Extract name from URL
        url.split('/')
            .filter(|s| !s.is_empty())
            .last()
            .unwrap_or("custom")
            .to_string()
    });
    
    // Check if already exists
    if config.repositories.iter().any(|r| r.name == repo_name) {
        println!("{} Repository '{}' already exists", "✗".red(), repo_name);
        return Ok(());
    }
    
    config.repositories.push(crate::config::Repository {
        name: repo_name.clone(),
        url: url.to_string(),
        enabled: true,
    });
    
    config.save()?;
    
    println!("{} Repository '{}' added", "✓".green(), repo_name);
    println!("Run 'mix-pkg update' to sync the package database");
    
    Ok(())
}

fn remove_repo(name: &str) -> Result<()> {
    let mut config = Config::load()?;
    
    let initial_len = config.repositories.len();
    config.repositories.retain(|r| r.name != name);
    
    if config.repositories.len() == initial_len {
        println!("{} Repository '{}' not found", "✗".red(), name);
        return Ok(());
    }
    
    config.save()?;
    
    println!("{} Repository '{}' removed", "✓".green(), name);
    
    Ok(())
}
