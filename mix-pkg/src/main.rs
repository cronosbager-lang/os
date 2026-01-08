mod cli;
mod config;
mod core;
mod utils;

use clap::Parser;
use cli::Cli;

fn main() {
    let cli = Cli::parse();
    
    if let Err(e) = cli::run(cli) {
        eprintln!("\x1b[31m✗\x1b[0m Error: {}", e);
        std::process::exit(1);
    }
}
