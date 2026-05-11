use std::path::Path;
use std::process::Command;
use crate::messages::{Finding, Severity, Location};
use serde::Deserialize;
use anyhow::{Context, Result};

#[derive(Debug, Deserialize)]
struct ClippyMessage {
    reason: String,
    message: Option<DiagMessage>,
}

#[derive(Debug, Deserialize)]
struct DiagMessage {
    level: String,
    message: String,
    code: Option<DiagCode>,
    spans: Vec<DiagSpan>,
}

#[derive(Debug, Deserialize)]
struct DiagCode {
    code: String,
}

#[derive(Debug, Deserialize)]
struct DiagSpan {
    file_name: String,
    line_start: u32,
    line_end: u32,
    is_primary: bool,
}

// Solana-specific Clippy lints and patterns that indicate security issues.
// These map clippy lint codes (or message patterns) to severity.
const SOLANA_HIGH_SEVERITY_LINTS: &[&str] = &[
    "clippy::integer_arithmetic",
    "clippy::cast_possible_truncation",
    "clippy::unwrap_used",
    "clippy::expect_used",
    "arithmetic_overflow",
];

const SOLANA_MEDIUM_SEVERITY_LINTS: &[&str] = &[
    "clippy::integer_division",
    "clippy::cast_sign_loss",
    "clippy::cast_possible_wrap",
    "clippy::indexing_slicing",
    "clippy::panic",
];

pub fn run(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running clippy (Solana) on {:?}", source_dir);

    let output = Command::new("cargo")
        .args([
            "clippy",
            "--message-format=json",
            "--",
            "-W", "clippy::integer_arithmetic",
            "-W", "clippy::cast_possible_truncation",
            "-W", "clippy::unwrap_used",
            "-W", "clippy::indexing_slicing",
            "-W", "clippy::integer_division",
            "-W", "clippy::cast_sign_loss",
            "-W", "clippy::panic",
        ])
        .current_dir(source_dir)
        .output()
        .context("Failed to execute cargo clippy")?;

    let stdout = String::from_utf8_lossy(&output.stdout);
    let mut findings = Vec::new();
    let mut idx = 0usize;

    for line in stdout.lines() {
        let msg: ClippyMessage = match serde_json::from_str(line) {
            Ok(m) => m,
            Err(_) => continue,
        };

        if msg.reason != "compiler-message" {
            continue;
        }

        let diag = match msg.message {
            Some(d) if d.level == "warning" || d.level == "error" => d,
            _ => continue,
        };

        let lint_code = diag.code.as_ref().map(|c| c.code.as_str()).unwrap_or("");

        let severity = if diag.level == "error" || SOLANA_HIGH_SEVERITY_LINTS.iter().any(|l| lint_code.contains(l)) {
            Severity::High
        } else if SOLANA_MEDIUM_SEVERITY_LINTS.iter().any(|l| lint_code.contains(l)) {
            Severity::Medium
        } else {
            Severity::Low
        };

        let primary_span = diag.spans.iter().find(|s| s.is_primary);
        let location = primary_span.map(|s| Location {
            file: s.file_name.clone(),
            line_start: s.line_start,
            line_end: s.line_end,
        }).unwrap_or_default();

        findings.push(Finding {
            id: format!("{}-clippy-{}", target_id, idx),
            target_id: target_id.to_string(),
            bounty_type: Some("solana-program".to_string()),
            category: format!("clippy:{}", lint_code),
            severity,
            tools: vec!["clippy".to_string()],
            location,
            description: diag.message,
        });
        idx += 1;
    }

    tracing::info!("clippy found {} issues", findings.len());
    Ok(findings)
}
