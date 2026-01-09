mod install;
mod remove;
mod search;
mod update;
mod repo;

use clap::{Parser, Subcommand};
use anyhow::Result;

#[derive(Parser)]
#[clap(name = "mix-pkg")]
#[clap(author = "MIXOS Team")]
#[clap(version = "1.0.0")]
#[clap(about = "MIXOS Package Manager", long_about = None)]
pub struct Cli {
    #[clap(subcommand)]
    pub command: Commands,

    /// Auto-confirm all prompts
    #[clap(short, long, global = true)]
    pub yes: bool,

    /// Verbose output
    #[clap(short, long, global = true)]
    pub verbose: bool,
}

#[derive(Subcommand)]
pub enum Commands {
    /// Install packages
    Install {
        /// Package names to install
        packages: Vec<String>,
        
        /// Force reinstallation
        #[clap(long)]
        force: bool,
    },
    
    /// Remove packages
    Remove {
        /// Package names to remove
        packages: Vec<String>,
    },
    
    /// Search for packages
    Search {
        /// Search query
        query: String,
    },
    
    /// Show package information
    Info {
        /// Package name
        package: String,
    },
    
    /// List installed packages
    List,
    
    /// Update package database
    Update,
    
    /// Upgrade all packages
    Upgrade,
    
    /// Repository management
    Repo {
        #[clap(subcommand)]
        command: RepoCommands,
    },
    
    /// Clean package cache
    Clean,
}

#[derive(Subcommand)]
pub enum RepoCommands {
    /// List repositories
    List,
    /// Add a repository
    Add {
        /// Repository URL
        url: String,
        /// Repository name
        #[clap(short, long)]
        name: Option<String>,
    },
    /// Remove a repository
    Remove {
        /// Repository name
        name: String,
    },
}

pub fn run(cli: Cli) -> Result<()> {
    match cli.command {
        Commands::Install { packages, force } => {
            install::run(&packages, force, cli.yes, cli.verbose)
        }
        Commands::Remove { packages } => {
            remove::run(&packages, cli.yes, cli.verbose)
        }
        Commands::Search { query } => {
            search::run_search(&query)
        }
        Commands::Info { package } => {
            search::run_info(&package)
        }
        Commands::List => {
            search::run_list()
        }
        Commands::Update => {
            update::run_update(cli.verbose)
        }
        Commands::Upgrade => {
            update::run_upgrade(cli.yes, cli.verbose)
        }
        Commands::Repo { command } => {
            repo::run(command)
        }
        Commands::Clean => {
            clean_cache()
        }
    }
}

fn clean_cache() -> Result<()> {
    use colored::Colorize;
    use std::fs;
    
    let cache_dir = dirs::cache_dir()
        .unwrap_or_else(|| std::path::PathBuf::from("/var/cache"))
        .join("mix-pkg");
    
    if cache_dir.exists() {
        fs::remove_dir_all(&cache_dir)?;
        fs::create_dir_all(&cache_dir)?;
    }
    
    println!("{} Package cache cleaned", "✓".green());
    Ok(())
}
