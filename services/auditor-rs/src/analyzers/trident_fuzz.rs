use std::path::Path;
use std::process::Command;
use crate::messages::{Finding, Severity, Location};
use anyhow::{Context, Result};

/// Maximum time to run the fuzzer per target (seconds). Kept short for CI.
/// Set TRIDENT_FUZZ_SECS env var to override.
const DEFAULT_FUZZ_SECS: u64 = 30;

pub fn run(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running Trident fuzzer on {:?}", source_dir);

    // Trident requires an Anchor workspace. Check for Anchor.toml first.
    if !source_dir.join("Anchor.toml").exists() {
        tracing::info!("trident: no Anchor.toml found, skipping (not an Anchor program)");
        return Ok(vec![]);
    }

    // Initialize Trident if no fuzz directory exists yet
    let fuzz_dir = source_dir.join("trident-tests");
    if !fuzz_dir.exists() {
        tracing::info!("trident: initializing fuzz workspace");
        let init = Command::new("trident")
            .args(["init"])
            .current_dir(source_dir)
            .output()
            .context("Failed to run `trident init` (install with `cargo install trident-cli`)")?;

        if !init.status.success() {
            tracing::warn!("trident init failed: {}", String::from_utf8_lossy(&init.stderr));
            return Ok(vec![]);
        }
    }

    let fuzz_secs = std::env::var("TRIDENT_FUZZ_SECS")
        .ok()
        .and_then(|s| s.parse().ok())
        .unwrap_or(DEFAULT_FUZZ_SECS);

    tracing::info!("trident: fuzzing for {} seconds", fuzz_secs);

    // Run the fuzzer — Trident wraps cargo-fuzz / libFuzzer under the hood
    let output = Command::new("trident")
        .args([
            "fuzz",
            "run-hfuzz",          // honggfuzz backend (faster for Solana)
            "fuzz_instructions",  // default target name from `trident init`
            "--",
            &format!("-t{}", fuzz_secs),  // honggfuzz timeout flag
            "-n1",                         // single thread (deterministic)
        ])
        .current_dir(source_dir)
        .output()
        .context("Failed to run `trident fuzz`")?;

    let combined = format!(
        "{}\n{}",
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr),
    );

    parse_trident_output(&combined, target_id, fuzz_secs)
}

fn parse_trident_output(output: &str, target_id: &str, fuzz_secs: u64) -> Result<Vec<Finding>> {
    let mut findings = Vec::new();
    let lower = output.to_lowercase();

    // Crash indicators from honggfuzz / libFuzzer output
    let is_crash = lower.contains("crash")
        || lower.contains("sigsegv")
        || lower.contains("sigabrt")
        || lower.contains("heap-buffer-overflow")
        || lower.contains("stack-overflow")
        || lower.contains("use-after-free")
        || lower.contains("program error")
        || output.contains("UNIQUE_CRASHES");

    // Panic in the Solana program (transaction-level)
    let is_panic = lower.contains("panicked at")
        || lower.contains("called `option::unwrap()` on a `none`")
        || lower.contains("called `result::unwrap()` on an `err`")
        || lower.contains("attempt to subtract with overflow")
        || lower.contains("attempt to add with overflow")
        || lower.contains("attempt to multiply with overflow");

    // Custom error from program (may indicate unexpected state)
    let is_custom_error = lower.contains("custom program error")
        && !lower.contains("expected error");

    // Extract crash file path if present
    let crash_file = extract_crash_file(output);
    let fuzz_location = crash_file.as_deref().unwrap_or("trident-tests/");

    if is_crash {
        findings.push(Finding {
            id: format!("{}-trident-crash", target_id),
            target_id: target_id.to_string(),
            bounty_type: Some("solana-program".to_string()),
            category: "trident:crash".to_string(),
            severity: Severity::Critical,
            tools: vec!["trident".to_string(), "honggfuzz".to_string()],
            location: Location {
                file: fuzz_location.to_string(),
                line_start: 0,
                line_end: 0,
            },
            description: format!(
                "Trident fuzzer triggered a crash after {}s. Crash input saved to: {}. Output: {}",
                fuzz_secs,
                fuzz_location,
                truncate(output, 600),
            ),
        });
    }

    if is_panic && !is_crash {
        findings.push(Finding {
            id: format!("{}-trident-panic", target_id),
            target_id: target_id.to_string(),
            bounty_type: Some("solana-program".to_string()),
            category: "trident:panic".to_string(),
            severity: Severity::High,
            tools: vec!["trident".to_string()],
            location: Location {
                file: "trident-tests/".to_string(),
                line_start: 0,
                line_end: 0,
            },
            description: format!(
                "Trident fuzzer triggered a panic in on-chain code. {}",
                extract_panic_line(output),
            ),
        });
    }

    if is_custom_error && !is_crash && !is_panic {
        findings.push(Finding {
            id: format!("{}-trident-error", target_id),
            target_id: target_id.to_string(),
            bounty_type: Some("solana-program".to_string()),
            category: "trident:unexpected-error".to_string(),
            severity: Severity::Medium,
            tools: vec!["trident".to_string()],
            location: Location {
                file: "trident-tests/".to_string(),
                line_start: 0,
                line_end: 0,
            },
            description: format!(
                "Trident fuzzer triggered an unexpected custom program error — may indicate reachable error paths. {}",
                truncate(output, 300),
            ),
        });
    }

    if findings.is_empty() {
        tracing::info!("trident: clean run ({} seconds, no crashes)", fuzz_secs);
    } else {
        tracing::warn!("trident: {} findings", findings.len());
    }

    Ok(findings)
}

fn extract_crash_file(output: &str) -> Option<String> {
    for line in output.lines() {
        if line.contains("CRASH") || line.contains("crash") {
            if let Some(path) = line.split_whitespace().find(|s| s.contains("trident") || s.ends_with(".fuzz")) {
                return Some(path.to_string());
            }
        }
    }
    None
}

fn extract_panic_line(output: &str) -> String {
    for line in output.lines() {
        if line.contains("panicked at") {
            return line.trim().to_string();
        }
    }
    truncate(output, 200)
}

fn truncate(s: &str, max: usize) -> String {
    if s.len() <= max {
        s.to_string()
    } else {
        format!("{}…", &s[..max])
    }
}
