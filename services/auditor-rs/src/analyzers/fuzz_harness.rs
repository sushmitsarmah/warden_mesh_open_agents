use std::path::Path;
use std::process::Command;
use crate::messages::{Finding, Severity, Location};
use anyhow::{Context, Result};

/// Accepted solfuzz-agave harness targets per scope.md:24
const ACCEPTED_HARNESSES: &[&str] = &[
    "instr_execute",
    "txn_execute",
    "elf_loader",
    "vm_interp",
    "vm_syscall_execute",
    "shred_parse",
    "pack_compute_budget",
];

pub fn run(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running fuzz harnesses on {:?}", source_dir);

    let mut findings = Vec::new();

    // Build firedancer first
    tracing::info!("Building firedancer with GCC-14...");
    let build_result = Command::new("make")
        .args(&["-j", "$(nproc)"])
        .current_dir(source_dir)
        .env("CC", "gcc-14")
        .env("CXX", "g++-14")
        .output()
        .context("Failed to build firedancer")?;

    if !build_result.status.success() {
        let stderr = String::from_utf8_lossy(&build_result.stderr);
        tracing::warn!("firedancer build failed, trying without compiler override: {}", stderr);

        let build_result = Command::new("make")
            .args(&["-j", "$(nproc)"])
            .current_dir(source_dir)
            .output();

        match build_result {
            Ok(out) if out.status.success() => {
                tracing::info!("firedancer build succeeded (default compiler)");
            }
            _ => {
                tracing::warn!("firedancer build failed entirely, skipping fuzz harnesses");
                return Ok(findings);
            }
        }
    } else {
        tracing::info!("firedancer build succeeded with GCC-14");
    }

    // Check for agave binary (need for bank hash comparison)
    let _has_agave = Command::new("which")
        .arg("agave-validator")
        .output()
        .map(|o| o.status.success())
        .unwrap_or(false);

    // Run each accepted harness
    for harness in ACCEPTED_HARNESSES {
        tracing::info!("Running fuzz harness: {}", harness);

        let harness_binary = source_dir.join("build").join(harness);
        if !harness_binary.exists() {
            // Try finding it elsewhere
            let find_result = Command::new("find")
                .args(&[source_dir.to_str().unwrap_or("."), "-name", harness, "-type", "f"])
                .output();

            match find_result {
                Ok(out) => {
                    let output_str = String::from_utf8_lossy(&out.stdout);
                    if output_str.is_empty() {
                        tracing::warn!("fuzz harness {} not found, skipping", harness);
                        continue;
                    }
                }
                Err(_) => {
                    tracing::warn!("failed to find fuzz harness {}", harness);
                    continue;
                }
            }
        }

        // Run the harness with a minimal test input
        let harness_result = Command::new(&harness_binary)
            .args(&["--benchmark", "--duration", "5"])
            .current_dir(source_dir)
            .output();

        match harness_result {
            Ok(out) => {
                let stdout = String::from_utf8_lossy(&out.stdout);
                let stderr = String::from_utf8_lossy(&out.stderr);
                let combined = format!("{}\n{}", stdout, stderr);

                // Check for crash indicators
                let is_crash = !out.status.success()
                    || combined.contains("SIGSEGV")
                    || combined.contains("SIGABRT")
                    || combined.contains("SEGFAULT")
                    || combined.contains("abort")
                    || combined.contains("panic");

                // Check for bank hash mismatch (compare Firedancer vs Agave)
                let is_bank_hash_mismatch = combined.contains("bank hash mismatch")
                    || combined.contains("BankHashMismatch")
                    || combined.contains("MISMATCH");

                // Reject error-code-only crashes (not eligible per scope.md:24)
                let is_error_code_only = combined.contains("error code mismatch")
                    || combined.contains("ErrorCode")
                    || combined.contains("result: Err");

                if is_crash && is_error_code_only {
                    tracing::warn!("harness {}: error-code-only crash (not eligible)", harness);
                    continue;
                }

                if is_crash || is_bank_hash_mismatch {
                    let category = if is_bank_hash_mismatch {
                        "bank-hash-mismatch"
                    } else {
                        "crash"
                    };

                    let description = format!(
                        "Harness {} triggered {}: {}",
                        harness,
                        category,
                        truncate(&combined, 500)
                    );

                    findings.push(Finding {
                        id: format!("{}-fuzz-{:.6}", target_id, harness.chars().take(16).collect::<String>()),
                        target_id: target_id.to_string(),
                        bounty_type: None,
                        category: category.to_string(),
                        severity: Severity::Critical,
                        tools: vec!["fuzz-harness".to_string()],
                        location: Location {
                            file: format!("harness/{}", harness),
                            line_start: 0,
                            line_end: 0,
                        },
                        description,
                    });

                    tracing::info!("fuzz harness {}: FOUND {}!", harness, category);
                } else if !out.status.success() {
                    tracing::warn!("harness {} failed without eligible impact: {}", harness, combined.lines().next().unwrap_or(""));
                } else {
                    tracing::info!("harness {}: clean run", harness);
                }
            }
            Err(e) => {
                tracing::warn!("harness {} execution failed: {}", harness, e);
            }
        }
    }

    Ok(findings)
}

fn truncate(s: &str, max: usize) -> String {
    if s.len() <= max {
        s.to_string()
    } else {
        format!("{}...", &s[..max])
    }
}
