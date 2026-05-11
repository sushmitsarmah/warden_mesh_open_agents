use std::path::Path;
use std::process::Command;
use std::fs;
use crate::messages::{Finding, Severity, Location};
use anyhow::{Context, Result};

// Default deny.toml with Solana-relevant checks enabled if none exists
const DEFAULT_DENY_TOML: &str = r#"
[advisories]
vulnerability = "deny"
unmaintained = "warn"
yanked = "deny"
notice = "warn"

[licenses]
unlicensed = "deny"
copyleft = "warn"
allow = ["MIT", "Apache-2.0", "ISC", "BSD-2-Clause", "BSD-3-Clause", "Unicode-DFS-2016"]

[bans]
multiple-versions = "warn"
wildcards = "deny"

[sources]
unknown-registry = "deny"
unknown-git = "warn"
"#;

pub fn run(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running cargo-deny on {:?}", source_dir);

    // Write a default deny.toml if the project doesn't have one
    let deny_toml = source_dir.join("deny.toml");
    let wrote_default = if !deny_toml.exists() {
        fs::write(&deny_toml, DEFAULT_DENY_TOML)?;
        true
    } else {
        false
    };

    let output = Command::new("cargo")
        .args([
            "deny",
            "--format", "json",
            "check",
            "advisories",
            "bans",
            "licenses",
            "sources",
        ])
        .current_dir(source_dir)
        .output()
        .context("Failed to execute cargo deny (install with `cargo install cargo-deny`)")?;

    // Clean up the synthetic deny.toml we created
    if wrote_default {
        let _ = fs::remove_file(&deny_toml);
    }

    // cargo deny exits non-zero when violations are found — that's expected
    let stdout = String::from_utf8_lossy(&output.stdout);
    let stderr = String::from_utf8_lossy(&output.stderr);

    // cargo deny --format json writes NDJSON to stderr (one diagnostic per line)
    let diagnostic_source = if stdout.trim().is_empty() { &stderr } else { &stdout };

    parse_deny_output(diagnostic_source, target_id)
}

fn parse_deny_output(output: &std::borrow::Cow<str>, target_id: &str) -> Result<Vec<Finding>> {
    let mut findings = Vec::new();
    let mut idx = 0usize;

    for line in output.lines() {
        let line = line.trim();
        if line.is_empty() { continue; }

        let diag: serde_json::Value = match serde_json::from_str(line) {
            Ok(v) => v,
            Err(_) => continue,
        };

        let level = diag.get("type")
            .and_then(|v| v.as_str())
            .or_else(|| diag.get("level").and_then(|v| v.as_str()))
            .unwrap_or("warning");

        if level == "summary" || level == "note" {
            continue;
        }

        let message = diag.get("message")
            .and_then(|v| v.as_str())
            .or_else(|| diag.get("fields").and_then(|f| f.get("message")).and_then(|v| v.as_str()))
            .unwrap_or("cargo-deny violation")
            .to_string();

        let fields = diag.get("fields").cloned().unwrap_or_default();
        let category = fields.get("category")
            .and_then(|v| v.as_str())
            .or_else(|| diag.get("category").and_then(|v| v.as_str()))
            .unwrap_or("dependency-policy")
            .to_string();

        let crate_name = fields.get("crate")
            .and_then(|v| v.as_str())
            .or_else(|| diag.get("crate_name").and_then(|v| v.as_str()))
            .unwrap_or("unknown")
            .to_string();

        let severity = match level {
            "error" | "deny" => match category.as_str() {
                "advisories" => Severity::High,    // known CVE in dependency
                "sources"    => Severity::Medium,  // untrusted registry
                "bans"       => Severity::Low,     // policy violation
                _            => Severity::Low,
            },
            "warning" | "warn" => Severity::Info,
            _ => Severity::Info,
        };

        findings.push(Finding {
            id: format!("{}-cargo-deny-{}", target_id, idx),
            target_id: target_id.to_string(),
            bounty_type: Some("solana-program".to_string()),
            category: format!("cargo-deny:{}", category),
            severity,
            tools: vec!["cargo-deny".to_string()],
            location: Location {
                file: format!("Cargo.toml ({})", crate_name),
                line_start: 0,
                line_end: 0,
            },
            description: message,
        });
        idx += 1;
    }

    tracing::info!("cargo-deny: {} findings", findings.len());
    Ok(findings)
}
