//! MRM Export Binary
//!
//! Export trained models to GGUF format.

use std::path::PathBuf;

use clap::Parser;
use log::info;

use mrm::MrmConfig;
use mrm::inference::ResonanceField;

/// MRM Export CLI
#[derive(Parser, Debug)]
#[command(name = "mrm-export")]
#[command(about = "Export MRM models to GGUF format")]
struct Args {
    /// Input model path (or "new" for fresh model)
    #[arg(short, long, default_value = "new")]
    input: String,
    
    /// Output GGUF file path
    #[arg(short, long)]
    output: PathBuf,
    
    /// Model configuration (for new models)
    #[arg(short, long)]
    config: Option<PathBuf>,
    
    /// Number of variants
    #[arg(long, default_value = "64")]
    variants: usize,
    
    /// Field resolution
    #[arg(long, default_value = "32")]
    resolution: usize,
    
    /// Verbose output
    #[arg(short, long)]
    verbose: bool,
}

fn main() -> anyhow::Result<()> {
    // Initialize logger
    env_logger::Builder::from_env(
        env_logger::Env::default().default_filter_or("info")
    ).init();
    
    let args = Args::parse();
    
    info!("MRM Export v2.0");
    info!("===============");
    
    // Create or load model
    let model = if args.input == "new" {
        info!("Creating new model");
        
        let config = if let Some(config_path) = &args.config {
            info!("Loading config from {:?}", config_path);
            let config_str = std::fs::read_to_string(config_path)?;
            serde_yaml::from_str(&config_str)?
        } else {
            let mut config = MrmConfig::default();
            config.variants.count = args.variants;
            config.field.resolution = args.resolution;
            config
        };
        
        info!("Configuration:");
        info!("  Variants: {}", config.variants.count);
        info!("  Domains: {}", config.variants.domains);
        info!("  Embed dim: {}", config.variants.embed_dim);
        info!("  Field resolution: {}", config.field.resolution);
        
        ResonanceField::with_config(config)
    } else {
        info!("Loading model from {}", args.input);
        mrm::io::gguf::import_model(&args.input)?
    };
    
    // Export to GGUF
    info!("Exporting to {:?}", args.output);
    
    // Create output directory if needed
    if let Some(parent) = args.output.parent() {
        std::fs::create_dir_all(parent)?;
    }
    
    mrm::io::gguf::export_model(&model, &args.output)?;
    
    // Print file info
    let metadata = std::fs::metadata(&args.output)?;
    let size_kb = metadata.len() as f64 / 1024.0;
    
    info!("Export complete!");
    info!("  File: {:?}", args.output);
    info!("  Size: {:.2} KB", size_kb);
    
    if args.verbose {
        let stats = model.stats();
        info!("Model Statistics:");
        info!("  Order parameter: {:.4}", stats.order_parameter);
        info!("  Mean mass: {:.4}", stats.mean_mass);
        info!("  Total energy: {:.4}", stats.total_energy);
    }
    
    Ok(())
}
