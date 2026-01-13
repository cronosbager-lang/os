//! MRM Training Binary
//!
//! Train the MixOS Recovery Model using Hebbian learning.

use std::path::PathBuf;

use clap::Parser;
use log::{info, warn, error};

use mrm::{MrmConfig, Input, Output};
use mrm::inference::ResonanceField;
use mrm::io::{DatasetLoader, GgufWriter};
use mrm::learning::compute_error;

/// MRM Training CLI
#[derive(Parser, Debug)]
#[command(name = "mrm-train")]
#[command(about = "Train the MixOS Recovery Model")]
struct Args {
    /// Path to training configuration file
    #[arg(short, long, default_value = "config/training/base.yaml")]
    config: PathBuf,
    
    /// Path to training data (JSONL)
    #[arg(short, long)]
    data: Option<PathBuf>,
    
    /// Output directory for model
    #[arg(short, long, default_value = "outputs")]
    output: PathBuf,
    
    /// Number of epochs
    #[arg(short, long, default_value = "100")]
    epochs: usize,
    
    /// Learning rate
    #[arg(short, long, default_value = "0.01")]
    learning_rate: f64,
    
    /// Batch size
    #[arg(short, long, default_value = "32")]
    batch_size: usize,
    
    /// Number of sample examples (when no data file provided)
    #[arg(short = 'n', long, default_value = "100")]
    num_samples: usize,
    
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
    
    info!("MRM Training v2.0");
    info!("================");
    
    // Load configuration
    let config = if args.config.exists() {
        info!("Loading config from {:?}", args.config);
        let config_str = std::fs::read_to_string(&args.config)?;
        serde_yaml::from_str(&config_str)?
    } else {
        info!("Using default configuration");
        MrmConfig::default()
    };
    
    // Create model
    info!("Creating ResonanceField with {} variants", config.variants.count);
    let mut model = ResonanceField::with_config(config.clone());
    
    // Load training data
    let mut loader = DatasetLoader::new();
    
    if let Some(data_path) = &args.data {
        info!("Loading training data from {:?}", data_path);
        let count = loader.load_jsonl(data_path)?;
        info!("Loaded {} examples", count);
    } else {
        // Use sample data for demonstration
        warn!("No training data provided, using {} sample examples", args.num_samples);
        for _ in 0..args.num_samples {
            let example = mrm::io::dataset::create_sample_example();
            loader.examples.push(example);
        }
    }
    
    // Convert to input/output pairs
    let pairs = loader.to_input_output_pairs()?;
    info!("Prepared {} training pairs", pairs.len());
    
    // Training loop
    info!("Starting training for {} epochs", args.epochs);
    info!("Learning rate: {}", args.learning_rate);
    info!("Batch size: {}", args.batch_size);
    
    let mut best_accuracy = 0.0;
    
    for epoch in 0..args.epochs {
        let mut epoch_error = 0.0;
        let mut epoch_count = 0;
        
        // Process in batches
        for batch in pairs.chunks(args.batch_size) {
            for (input, expected) in batch {
                // Run inference
                let actual = model.infer(input);
                
                // Compute error
                let error = compute_error(&actual, expected);
                epoch_error += error;
                epoch_count += 1;
                
                // Learn from example
                model.learn(input, expected, args.learning_rate);
            }
        }
        
        let avg_error = epoch_error / epoch_count as f64;
        let accuracy = 1.0 - avg_error;
        
        if args.verbose || epoch % 10 == 0 {
            let stats = model.stats();
            info!(
                "Epoch {}/{}: accuracy={:.4}, error={:.4}, order_param={:.4}, coherence={:.4}",
                epoch + 1, args.epochs, accuracy, avg_error,
                stats.order_parameter, stats.coherence
            );
        }
        
        // Save best model
        if accuracy > best_accuracy {
            best_accuracy = accuracy;
            
            let model_path = args.output.join("best_model.gguf");
            std::fs::create_dir_all(&args.output)?;
            
            if let Err(e) = mrm::io::gguf::export_model(&model, &model_path) {
                warn!("Failed to save model: {}", e);
            } else if args.verbose {
                info!("Saved best model (accuracy={:.4})", accuracy);
            }
        }
    }
    
    // Final save
    let final_path = args.output.join("final_model.gguf");
    mrm::io::gguf::export_model(&model, &final_path)?;
    
    info!("Training complete!");
    info!("Best accuracy: {:.4}", best_accuracy);
    info!("Final model saved to {:?}", final_path);
    
    // Print final statistics
    let stats = model.stats();
    info!("Final Statistics:");
    info!("  Order parameter: {:.4}", stats.order_parameter);
    info!("  Coherence: {:.4}", stats.coherence);
    info!("  Total energy: {:.4}", stats.total_energy);
    info!("  Mean mass: {:.4}", stats.mean_mass);
    info!("  Mean resonance: {:.4}", stats.mean_resonance);
    
    Ok(())
}
