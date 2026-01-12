//! MRM Inference Binary
//!
//! Run inference on error inputs using the trained model.

use std::path::PathBuf;
use std::io::{self, BufRead, Write};

use clap::Parser;
use log::{info, warn, error};

use mrm::{Input, Context, SystemState, BootStage};
use mrm::inference::ResonanceField;
use mrm::safety::SafetyGuard;

/// MRM Inference CLI
#[derive(Parser, Debug)]
#[command(name = "mrm-infer")]
#[command(about = "Run MRM inference on error inputs")]
struct Args {
    /// Path to model file (GGUF)
    #[arg(short, long)]
    model: Option<PathBuf>,
    
    /// Error message to diagnose
    #[arg(short, long)]
    error: Option<String>,
    
    /// Boot stage (bootloader, kernel, init, services, ready)
    #[arg(short, long, default_value = "init")]
    stage: String,
    
    /// Interactive mode (read from stdin)
    #[arg(short, long)]
    interactive: bool,
    
    /// Output format (text, json)
    #[arg(short, long, default_value = "text")]
    format: String,
    
    /// Apply safety checks
    #[arg(long, default_value = "true")]
    safety: bool,
    
    /// Verbose output
    #[arg(short, long)]
    verbose: bool,
}

fn main() -> anyhow::Result<()> {
    // Initialize logger
    let log_level = if std::env::args().any(|a| a == "-v" || a == "--verbose") {
        "debug"
    } else {
        "warn"
    };
    env_logger::Builder::from_env(
        env_logger::Env::default().default_filter_or(log_level)
    ).init();
    
    let args = Args::parse();
    
    // Load or create model
    let mut model = if let Some(model_path) = &args.model {
        info!("Loading model from {:?}", model_path);
        mrm::io::gguf::import_model(model_path)?
    } else {
        info!("Using default model");
        ResonanceField::new()
    };
    
    // Create safety guard
    let safety = if args.safety {
        SafetyGuard::new()
    } else {
        SafetyGuard::permissive()
    };
    
    if args.interactive {
        run_interactive(&mut model, &safety, &args)?;
    } else if let Some(error_msg) = &args.error {
        run_single(&mut model, &safety, error_msg, &args)?;
    } else {
        eprintln!("Error: Provide --error or use --interactive mode");
        std::process::exit(1);
    }
    
    Ok(())
}

fn run_single(
    model: &mut ResonanceField,
    safety: &SafetyGuard,
    error_msg: &str,
    args: &Args,
) -> anyhow::Result<()> {
    let input = create_input(error_msg, &args.stage);
    let output = model.infer(&input);
    
    // Apply safety checks
    let safe_output = if args.safety {
        safety.filter_safe(&output)
    } else {
        output.clone()
    };
    
    // Output result
    match args.format.as_str() {
        "json" => {
            println!("{}", serde_json::to_string_pretty(&safe_output)?);
        }
        _ => {
            print_output(&safe_output, args.verbose);
        }
    }
    
    Ok(())
}

fn run_interactive(
    model: &mut ResonanceField,
    safety: &SafetyGuard,
    args: &Args,
) -> anyhow::Result<()> {
    println!("MRM Interactive Mode");
    println!("====================");
    println!("Enter error messages (Ctrl+D to exit)");
    println!();
    
    let stdin = io::stdin();
    let mut stdout = io::stdout();
    
    loop {
        print!("> ");
        stdout.flush()?;
        
        let mut line = String::new();
        if stdin.lock().read_line(&mut line)? == 0 {
            break;
        }
        
        let error_msg = line.trim();
        if error_msg.is_empty() {
            continue;
        }
        
        // Handle commands
        if error_msg.starts_with(':') {
            match error_msg {
                ":quit" | ":q" => break,
                ":stats" => {
                    let stats = model.stats();
                    println!("Order parameter: {:.4}", stats.order_parameter);
                    println!("Coherence: {:.4}", stats.coherence);
                    println!("Total energy: {:.4}", stats.total_energy);
                    continue;
                }
                ":reset" => {
                    model.reset();
                    println!("Model reset");
                    continue;
                }
                ":help" => {
                    println!("Commands:");
                    println!("  :quit, :q  - Exit");
                    println!("  :stats     - Show model statistics");
                    println!("  :reset     - Reset model state");
                    println!("  :help      - Show this help");
                    continue;
                }
                _ => {
                    println!("Unknown command. Type :help for help.");
                    continue;
                }
            }
        }
        
        let input = create_input(error_msg, &args.stage);
        let output = model.infer(&input);
        
        let safe_output = if args.safety {
            safety.filter_safe(&output)
        } else {
            output
        };
        
        println!();
        print_output(&safe_output, args.verbose);
        println!();
    }
    
    println!("Goodbye!");
    Ok(())
}

fn create_input(error_msg: &str, stage: &str) -> Input {
    let boot_stage = match stage.to_lowercase().as_str() {
        "bootloader" => BootStage::Bootloader,
        "kernel" => BootStage::Kernel,
        "init" => BootStage::Init,
        "services" => BootStage::Services,
        "ready" => BootStage::Ready,
        _ => BootStage::Init,
    };
    
    Input {
        error: error_msg.to_string(),
        context: Context {
            kernel_version: Some("6.1.0-mixos".to_string()),
            boot_stage,
            last_action: None,
            uptime_seconds: Some(10),
        },
        state: SystemState {
            memory_available: true,
            root_mounted: boot_stage != BootStage::Bootloader && boot_stage != BootStage::Kernel,
            network_up: boot_stage == BootStage::Ready,
            services_started: vec![],
        },
    }
}

fn print_output(output: &mrm::Output, verbose: bool) {
    println!("Diagnosis: {}", output.diagnosis);
    
    if let Some(cause) = &output.root_cause {
        println!("Root Cause: {}", cause);
    }
    
    println!("Confidence: {:.2}%", output.confidence * 100.0);
    
    if verbose {
        println!("Resonance Coherence: {:.4}", output.resonance_coherence);
    }
    
    println!();
    println!("Recommended Actions:");
    for action in &output.actions {
        println!("  {}. {} ", action.order, action.action);
        if verbose {
            println!("     Params: {}", action.params);
        }
        if let Some(fallback) = &action.fallback {
            println!("     Fallback: {}", fallback);
        }
    }
}
