//! MRM Benchmark Binary
//!
//! Run comprehensive benchmarks and generate reports.

use std::path::PathBuf;

use clap::Parser;
use log::info;

use mrm::benchmark::{BenchmarkRunner, BenchmarkReport};
use mrm::inference::ResonanceField;
use mrm::io::DatasetLoader;

/// MRM Benchmark CLI
#[derive(Parser, Debug)]
#[command(name = "mrm-benchmark")]
#[command(about = "Run MRM benchmarks and generate reports")]
struct Args {
    /// Path to model file (GGUF)
    #[arg(short, long)]
    model: Option<PathBuf>,
    
    /// Path to test data (JSONL)
    #[arg(short, long)]
    data: Option<PathBuf>,
    
    /// Output directory for reports
    #[arg(short, long, default_value = "benchmark_results")]
    output: PathBuf,
    
    /// Number of sample examples (if no data file)
    #[arg(short = 'n', long, default_value = "100")]
    num_samples: usize,
    
    /// Verbose output
    #[arg(short, long)]
    verbose: bool,
}

fn main() -> anyhow::Result<()> {
    // Initialize logger
    let log_level = if std::env::args().any(|a| a == "-v" || a == "--verbose") {
        "info"
    } else {
        "warn"
    };
    env_logger::Builder::from_env(
        env_logger::Env::default().default_filter_or(log_level)
    ).init();
    
    let args = Args::parse();
    
    println!("🔬 MRM-64 Benchmark Suite");
    println!("========================\n");
    
    // Load or create model
    let model = if let Some(model_path) = &args.model {
        info!("Loading model from {:?}", model_path);
        println!("📦 Loading model: {:?}", model_path);
        mrm::io::gguf::import_model(model_path)?
    } else {
        info!("Using default model");
        println!("📦 Using default model");
        ResonanceField::new()
    };
    
    // Load test data
    let mut loader = DatasetLoader::new();
    
    if let Some(data_path) = &args.data {
        info!("Loading test data from {:?}", data_path);
        println!("📊 Loading test data: {:?}", data_path);
        let count = loader.load_jsonl(data_path)?;
        println!("   Loaded {} examples\n", count);
    } else {
        println!("📊 Generating {} sample examples\n", args.num_samples);
        for _ in 0..args.num_samples {
            let example = mrm::io::dataset::create_sample_example();
            loader.examples.push(example);
        }
    }
    
    // Run benchmark
    println!("🚀 Running benchmark...\n");
    
    let mut runner = BenchmarkRunner::new(model);
    let results = runner.run(&loader.examples);
    
    // Print summary
    println!("📈 Results Summary");
    println!("==================\n");
    
    println!("Overall Metrics:");
    println!("  Accuracy:     {:.2}%", results.overall.accuracy * 100.0);
    println!("  Macro F1:     {:.2}%", results.overall.macro_f1 * 100.0);
    println!("  Action F1:    {:.2}%", results.action.action_f1 * 100.0);
    println!();
    
    println!("Reasoning Quality:");
    println!("  Root Cause:   {:.2}%", results.reasoning.root_cause_accuracy * 100.0);
    println!("  Multi-step:   {:.2}%", results.reasoning.multi_step_accuracy * 100.0);
    println!("  Domain:       {:.2}%", results.reasoning.domain_accuracy * 100.0);
    println!();
    
    println!("Calibration:");
    println!("  ECE:          {:.4}", results.calibration.ece);
    println!("  Avg Conf:     {:.2}%", results.calibration.avg_confidence * 100.0);
    println!();
    
    println!("Performance:");
    println!("  Avg Latency:  {:.2}ms", results.performance.avg_latency_ms);
    println!("  P95 Latency:  {:.2}ms", results.performance.p95_latency_ms);
    println!("  Throughput:   {:.1} inf/s", results.performance.throughput);
    println!();
    
    println!("Field Dynamics:");
    println!("  Order Param:  {:.4}", results.field_dynamics.avg_order_parameter);
    println!("  Coherence:    {:.4}", results.field_dynamics.avg_coherence);
    println!();
    
    // Check targets
    if results.meets_target_goals() {
        println!("✅ MEETS TARGET GOALS!");
    } else if results.meets_minimum_targets() {
        println!("🟡 Meets minimum targets");
    } else {
        println!("🔴 Below minimum targets");
    }
    println!();
    
    // Generate reports
    println!("📝 Generating reports...");
    
    let report = BenchmarkReport::new(results);
    report.export_all(&args.output)?;
    
    println!("   ✅ benchmark_results.json");
    println!("   ✅ BENCHMARK.md");
    println!("   ✅ benchmark_section.md (for HuggingFace)");
    println!("   ✅ benchmark_metrics.csv");
    println!();
    println!("📁 Reports saved to: {:?}", args.output);
    
    Ok(())
}
