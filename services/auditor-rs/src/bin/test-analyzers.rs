use std::path::PathBuf;
use auditor::analyzers::{aderyn, slither, merge};

fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt::init();

    let contracts_dir = PathBuf::from("../../contracts");
    let target_id = "test-vault";

    println!("Testing analyzers on VulnerableVault...\n");

    // Run Aderyn
    println!("=== Running Aderyn ===");
    let aderyn_findings = aderyn::run_aderyn(&contracts_dir, target_id)?;
    println!("Aderyn found {} issues:", aderyn_findings.len());
    for finding in &aderyn_findings {
        println!("  [{:?}] {} at {}:{}",
            finding.severity,
            finding.category,
            finding.location.file,
            finding.location.line_start
        );
    }

    // Run Slither
    println!("\n=== Running Slither ===");
    let slither_findings = slither::run_slither(&contracts_dir, target_id)?;
    println!("Slither found {} issues:", slither_findings.len());
    for finding in &slither_findings {
        println!("  [{:?}] {} at {}:{}",
            finding.severity,
            finding.category,
            finding.location.file,
            finding.location.line_start
        );
    }

    // Merge findings
    println!("\n=== Merged Findings (dual-source filter) ===");
    let merged = merge::merge(aderyn_findings, slither_findings);
    println!("After merge: {} critical findings:", merged.len());
    for finding in &merged {
        println!("  [{:?}] {} (tools: {:?}) at {}:{}",
            finding.severity,
            finding.category,
            finding.tools,
            finding.location.file,
            finding.location.line_start
        );
        println!("    Description: {}", finding.description);
    }

    Ok(())
}
