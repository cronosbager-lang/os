//! Benchmark Report Generation
//!
//! Generates reports in multiple formats for HuggingFace and documentation.

use std::fs::File;
use std::io::Write;
use std::path::Path;

use serde_json;

use super::metrics::BenchmarkResults;

/// Benchmark report generator
pub struct BenchmarkReport {
    results: BenchmarkResults,
}

impl BenchmarkReport {
    /// Create new report from results
    pub fn new(results: BenchmarkResults) -> Self {
        Self { results }
    }

    /// Export all report formats
    pub fn export_all<P: AsRef<Path>>(&self, output_dir: P) -> std::io::Result<()> {
        let dir = output_dir.as_ref();
        std::fs::create_dir_all(dir)?;

        self.export_json(dir.join("benchmark_results.json"))?;
        self.export_markdown(dir.join("BENCHMARK.md"))?;
        self.export_hf_model_card_section(dir.join("benchmark_section.md"))?;
        self.export_csv(dir.join("benchmark_metrics.csv"))?;

        Ok(())
    }

    /// Export as JSON (full results)
    pub fn export_json<P: AsRef<Path>>(&self, path: P) -> std::io::Result<()> {
        let json = serde_json::to_string_pretty(&self.results)?;
        let mut file = File::create(path)?;
        file.write_all(json.as_bytes())?;
        Ok(())
    }

    /// Export as Markdown report
    pub fn export_markdown<P: AsRef<Path>>(&self, path: P) -> std::io::Result<()> {
        let mut content = String::new();

        // Header
        content.push_str("# MRM-64 Benchmark Results\n\n");
        content.push_str(&format!("**Model**: {} v{}\n", self.results.model_info.name, self.results.model_info.version));
        content.push_str(&format!("**Variants**: {}\n", self.results.model_info.variants));
        content.push_str(&format!("**Model Size**: {:.2} KB\n\n", self.results.model_info.model_size_bytes as f64 / 1024.0));

        // Status badges
        let status = if self.results.meets_target_goals() {
            "🟢 **MEETS TARGET GOALS**"
        } else if self.results.meets_minimum_targets() {
            "🟡 **MEETS MINIMUM TARGETS**"
        } else {
            "🔴 **BELOW MINIMUM TARGETS**"
        };
        content.push_str(&format!("{}\n\n", status));

        // Overall Metrics
        content.push_str("## Overall Metrics\n\n");
        content.push_str("| Metric | Value | Target | Minimum |\n");
        content.push_str("|--------|-------|--------|--------|\n");
        content.push_str(&format!("| Accuracy | {:.2}% | 97% | 95% |\n", self.results.overall.accuracy * 100.0));
        content.push_str(&format!("| Macro F1 | {:.2}% | - | - |\n", self.results.overall.macro_f1 * 100.0));
        content.push_str(&format!("| Action F1 | {:.2}% | 95% | 90% |\n", self.results.action.action_f1 * 100.0));
        content.push_str(&format!("| ECE | {:.4} | <0.05 | <0.10 |\n", self.results.calibration.ece));
        content.push_str(&format!("| Avg Latency | {:.1}ms | <30ms | <50ms |\n\n", self.results.performance.avg_latency_ms));

        // Reasoning Metrics
        content.push_str("## Reasoning Quality\n\n");
        content.push_str("| Metric | Value |\n");
        content.push_str("|--------|-------|\n");
        content.push_str(&format!("| Root Cause Accuracy | {:.2}% |\n", self.results.reasoning.root_cause_accuracy * 100.0));
        content.push_str(&format!("| Multi-step Accuracy | {:.2}% |\n", self.results.reasoning.multi_step_accuracy * 100.0));
        content.push_str(&format!("| Domain Accuracy | {:.2}% |\n", self.results.reasoning.domain_accuracy * 100.0));
        content.push_str(&format!("| Causal Accuracy | {:.2}% |\n", self.results.reasoning.causal_accuracy * 100.0));
        content.push_str(&format!("| Coherence Score | {:.4} |\n\n", self.results.reasoning.coherence_score));

        // Action Metrics
        content.push_str("## Action Prediction\n\n");
        content.push_str("| Metric | Value |\n");
        content.push_str("|--------|-------|\n");
        content.push_str(&format!("| Primary Action Accuracy | {:.2}% |\n", self.results.action.primary_action_accuracy * 100.0));
        content.push_str(&format!("| Sequence Accuracy | {:.2}% |\n", self.results.action.sequence_accuracy * 100.0));
        content.push_str(&format!("| Parameter Accuracy | {:.2}% |\n\n", self.results.action.param_accuracy * 100.0));

        // Per-action breakdown
        if !self.results.action.per_action.is_empty() {
            content.push_str("### Per-Action Performance\n\n");
            content.push_str("| Action | Precision | Recall | F1 | Support |\n");
            content.push_str("|--------|-----------|--------|----|---------|\n");
            
            let mut actions: Vec<_> = self.results.action.per_action.iter().collect();
            actions.sort_by(|a, b| b.1.support.cmp(&a.1.support));
            
            for (_, metrics) in actions.iter().take(10) {
                content.push_str(&format!(
                    "| {} | {:.2}% | {:.2}% | {:.2}% | {} |\n",
                    metrics.action,
                    metrics.precision * 100.0,
                    metrics.recall * 100.0,
                    metrics.f1 * 100.0,
                    metrics.support
                ));
            }
            content.push_str("\n");
        }

        // Calibration
        content.push_str("## Confidence Calibration\n\n");
        content.push_str("| Metric | Value |\n");
        content.push_str("|--------|-------|\n");
        content.push_str(&format!("| ECE (Expected Calibration Error) | {:.4} |\n", self.results.calibration.ece));
        content.push_str(&format!("| MCE (Maximum Calibration Error) | {:.4} |\n", self.results.calibration.mce));
        content.push_str(&format!("| Avg Confidence | {:.2}% |\n", self.results.calibration.avg_confidence * 100.0));
        content.push_str(&format!("| Confidence (Correct) | {:.2}% |\n", self.results.calibration.confidence_when_correct * 100.0));
        content.push_str(&format!("| Confidence (Incorrect) | {:.2}% |\n", self.results.calibration.confidence_when_incorrect * 100.0));
        content.push_str(&format!("| Overconfidence Rate | {:.2}% |\n", self.results.calibration.overconfidence_rate * 100.0));
        content.push_str(&format!("| Underconfidence Rate | {:.2}% |\n\n", self.results.calibration.underconfidence_rate * 100.0));

        // Calibration bins (reliability diagram data)
        if !self.results.calibration.calibration_bins.is_empty() {
            content.push_str("### Reliability Diagram Data\n\n");
            content.push_str("| Bin | Avg Confidence | Accuracy | Count |\n");
            content.push_str("|-----|----------------|----------|-------|\n");
            
            for bin in &self.results.calibration.calibration_bins {
                content.push_str(&format!(
                    "| {:.1}-{:.1} | {:.2}% | {:.2}% | {} |\n",
                    bin.bin_start * 100.0,
                    bin.bin_end * 100.0,
                    bin.avg_confidence * 100.0,
                    bin.accuracy * 100.0,
                    bin.count
                ));
            }
            content.push_str("\n");
        }

        // Performance
        content.push_str("## Performance\n\n");
        content.push_str("| Metric | Value |\n");
        content.push_str("|--------|-------|\n");
        content.push_str(&format!("| Avg Latency | {:.2}ms |\n", self.results.performance.avg_latency_ms));
        content.push_str(&format!("| P50 Latency | {:.2}ms |\n", self.results.performance.p50_latency_ms));
        content.push_str(&format!("| P95 Latency | {:.2}ms |\n", self.results.performance.p95_latency_ms));
        content.push_str(&format!("| P99 Latency | {:.2}ms |\n", self.results.performance.p99_latency_ms));
        content.push_str(&format!("| Throughput | {:.1} inf/s |\n", self.results.performance.throughput));
        content.push_str(&format!("| Memory Usage | {:.2} KB |\n\n", self.results.performance.memory_usage_bytes as f64 / 1024.0));

        // Field Dynamics
        content.push_str("## Field Dynamics (Kuramoto + Gravity)\n\n");
        content.push_str("| Metric | Value |\n");
        content.push_str("|--------|-------|\n");
        content.push_str(&format!("| Avg Order Parameter | {:.4} |\n", self.results.field_dynamics.avg_order_parameter));
        content.push_str(&format!("| Order Param (Correct) | {:.4} |\n", self.results.field_dynamics.order_param_when_correct));
        content.push_str(&format!("| Order Param (Incorrect) | {:.4} |\n", self.results.field_dynamics.order_param_when_incorrect));
        content.push_str(&format!("| Avg Coherence | {:.4} |\n", self.results.field_dynamics.avg_coherence));
        content.push_str(&format!("| Convergence Rate | {:.2}% |\n\n", self.results.field_dynamics.convergence_rate * 100.0));

        // Per-category
        if !self.results.per_category.is_empty() {
            content.push_str("## Per-Category Performance\n\n");
            content.push_str("| Category | Accuracy | F1 | Support |\n");
            content.push_str("|----------|----------|----|---------|\n");
            
            let mut categories: Vec<_> = self.results.per_category.iter().collect();
            categories.sort_by(|a, b| b.1.support.cmp(&a.1.support));
            
            for (_, metrics) in categories {
                content.push_str(&format!(
                    "| {} | {:.2}% | {:.2}% | {} |\n",
                    metrics.category,
                    metrics.accuracy * 100.0,
                    metrics.f1 * 100.0,
                    metrics.support
                ));
            }
            content.push_str("\n");
        }

        // Footer
        content.push_str("---\n\n");
        content.push_str(&format!("*Total Examples: {}*\n", self.results.overall.total_examples));

        let mut file = File::create(path)?;
        file.write_all(content.as_bytes())?;
        Ok(())
    }

    /// Export section for HuggingFace model card
    pub fn export_hf_model_card_section<P: AsRef<Path>>(&self, path: P) -> std::io::Result<()> {
        let mut content = String::new();

        content.push_str("## Benchmark Results\n\n");

        // Summary table
        content.push_str("### Performance Summary\n\n");
        content.push_str("| Metric | Value | Target |\n");
        content.push_str("|--------|-------|--------|\n");
        content.push_str(&format!("| **Accuracy** | {:.1}% | 97% |\n", self.results.overall.accuracy * 100.0));
        content.push_str(&format!("| **Action F1** | {:.1}% | 95% |\n", self.results.action.action_f1 * 100.0));
        content.push_str(&format!("| **ECE** | {:.4} | <0.05 |\n", self.results.calibration.ece));
        content.push_str(&format!("| **Latency** | {:.1}ms | <50ms |\n", self.results.performance.avg_latency_ms));
        content.push_str(&format!("| **Model Size** | {:.1}KB | <5MB |\n\n", self.results.model_info.model_size_bytes as f64 / 1024.0));

        // Reasoning
        content.push_str("### Reasoning Quality\n\n");
        content.push_str(&format!("- Root Cause Accuracy: {:.1}%\n", self.results.reasoning.root_cause_accuracy * 100.0));
        content.push_str(&format!("- Multi-step Accuracy: {:.1}%\n", self.results.reasoning.multi_step_accuracy * 100.0));
        content.push_str(&format!("- Domain Accuracy: {:.1}%\n\n", self.results.reasoning.domain_accuracy * 100.0));

        // Field dynamics highlight
        content.push_str("### Field Dynamics\n\n");
        content.push_str(&format!("- Order Parameter: {:.4} (sync level)\n", self.results.field_dynamics.avg_order_parameter));
        content.push_str(&format!("- Coherence: {:.4}\n", self.results.field_dynamics.avg_coherence));
        content.push_str(&format!("- Throughput: {:.0} inferences/sec\n\n", self.results.performance.throughput));

        let mut file = File::create(path)?;
        file.write_all(content.as_bytes())?;
        Ok(())
    }

    /// Export as CSV for analysis
    pub fn export_csv<P: AsRef<Path>>(&self, path: P) -> std::io::Result<()> {
        let mut content = String::new();

        content.push_str("metric,value\n");
        content.push_str(&format!("accuracy,{}\n", self.results.overall.accuracy));
        content.push_str(&format!("macro_f1,{}\n", self.results.overall.macro_f1));
        content.push_str(&format!("action_f1,{}\n", self.results.action.action_f1));
        content.push_str(&format!("ece,{}\n", self.results.calibration.ece));
        content.push_str(&format!("avg_latency_ms,{}\n", self.results.performance.avg_latency_ms));
        content.push_str(&format!("p95_latency_ms,{}\n", self.results.performance.p95_latency_ms));
        content.push_str(&format!("throughput,{}\n", self.results.performance.throughput));
        content.push_str(&format!("order_parameter,{}\n", self.results.field_dynamics.avg_order_parameter));
        content.push_str(&format!("coherence,{}\n", self.results.field_dynamics.avg_coherence));
        content.push_str(&format!("root_cause_accuracy,{}\n", self.results.reasoning.root_cause_accuracy));
        content.push_str(&format!("domain_accuracy,{}\n", self.results.reasoning.domain_accuracy));
        content.push_str(&format!("total_examples,{}\n", self.results.overall.total_examples));

        let mut file = File::create(path)?;
        file.write_all(content.as_bytes())?;
        Ok(())
    }

    /// Get results reference
    pub fn results(&self) -> &BenchmarkResults {
        &self.results
    }
}
