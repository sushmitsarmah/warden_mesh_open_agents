use std::path::Path;
use std::process::Command;
use crate::messages::{Finding, Severity, Location};
use anyhow::{Context, Result};

/// Unsafe block count above which we escalate from Low to Medium severity.
const UNSAFE_MEDIUM_THRESHOLD: u32 = 5;
/// Unsafe block count above which we escalate to High.
const UNSAFE_HIGH_THRESHOLD: u32 = 20;

pub fn run(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running cargo-geiger on {:?}", source_dir);

    // cargo geiger reports unsafe usage in the crate and all dependencies.
    // --output-format=Json gives machine-readable output.
    let output = Command::new("cargo")
        .args([
            "geiger",
            "--output-format", "Json",
            "--quiet",
        ])
        .current_dir(source_dir)
        .output()
        .context("Failed to execute cargo geiger (install with `cargo install cargo-geiger`)")?;

    let stdout = String::from_utf8_lossy(&output.stdout);
    if stdout.trim().is_empty() {
        tracing::info!("cargo-geiger: no output");
        return Ok(vec![]);
    }

    parse_geiger_output(&stdout, target_id)
}

fn parse_geiger_output(output: &str, target_id: &str) -> Result<Vec<Finding>> {
    let mut findings = Vec::new();

    // Geiger JSON is one JSON object per line (NDJSON) or a single object.
    // We do a best-effort parse: look for "unsafe_used" and "package" fields.
    let parsed: serde_json::Value = match serde_json::from_str(output) {
        Ok(v) => v,
        Err(_) => {
            // Try line-by-line NDJSON
            return parse_geiger_ndjson(output, target_id);
        }
    };

    // Top-level: { packages: [ { package: { name, version }, unsafety: { used: { exprs: { unsafe: N } } } } ] }
    if let Some(packages) = parsed.get("packages").and_then(|p| p.as_array()) {
        for (idx, pkg) in packages.iter().enumerate() {
            process_package(pkg, target_id, idx, &mut findings);
        }
    }

    tracing::info!("cargo-geiger: {} unsafe findings", findings.len());
    Ok(findings)
}

fn parse_geiger_ndjson(output: &str, target_id: &str) -> Result<Vec<Finding>> {
    let mut findings = Vec::new();
    let mut idx = 0usize;

    for line in output.lines() {
        let line = line.trim();
        if line.is_empty() { continue; }
        if let Ok(pkg) = serde_json::from_str::<serde_json::Value>(line) {
            process_package(&pkg, target_id, idx, &mut findings);
            idx += 1;
        }
    }

    tracing::info!("cargo-geiger (ndjson): {} unsafe findings", findings.len());
    Ok(findings)
}

fn process_package(pkg: &serde_json::Value, target_id: &str, idx: usize, findings: &mut Vec<Finding>) {
    let name = pkg.pointer("/package/name")
        .and_then(|v| v.as_str())
        .unwrap_or("unknown");
    let version = pkg.pointer("/package/version")
        .and_then(|v| v.as_str())
        .unwrap_or("?");
    let is_root = pkg.get("root").and_then(|v| v.as_bool()).unwrap_or(false);

    // Total unsafe expressions used (not just declared)
    let unsafe_count = pkg
        .pointer("/unsafety/used/exprs/unsafe")
        .and_then(|v| v.as_u64())
        .unwrap_or(0) as u32;

    if unsafe_count == 0 {
        return;
    }

    // Only report the root crate at High/Critical — deps at lower severity
    let severity = if is_root {
        if unsafe_count >= UNSAFE_HIGH_THRESHOLD {
            Severity::High
        } else if unsafe_count >= UNSAFE_MEDIUM_THRESHOLD {
            Severity::Medium
        } else {
            Severity::Low
        }
    } else {
        // Dependency with unsafe is informational unless count is very high
        if unsafe_count >= UNSAFE_HIGH_THRESHOLD {
            Severity::Medium
        } else {
            Severity::Info
        }
    };

    let context = if is_root { "root crate" } else { "dependency" };
    let description = format!(
        "{} {} v{} has {} unsafe expression(s) in use. \
        Unsafe code in on-chain programs can bypass Rust memory safety and introduce exploitable bugs. \
        Review all `unsafe` blocks for correct pointer arithmetic, raw memory access, and FFI calls.",
        context, name, version, unsafe_count
    );

    findings.push(Finding {
        id: format!("{}-geiger-{}", target_id, idx),
        target_id: target_id.to_string(),
        bounty_type: Some("solana-program".to_string()),
        category: "unsafe-code".to_string(),
        severity,
        tools: vec!["cargo-geiger".to_string()],
        location: Location {
            file: format!("Cargo.toml ({})", name),
            line_start: 0,
            line_end: 0,
        },
        description,
    });
}
