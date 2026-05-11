use std::path::Path;
use std::process::Command;
use crate::messages::{Finding, Severity, Location};
use serde::Deserialize;
use anyhow::{Context, Result};

#[derive(Debug, Deserialize)]
struct SemgrepOutput {
    results: Vec<SemgrepResult>,
    #[serde(default)]
    errors: Vec<serde_json::Value>,
}

#[derive(Debug, Deserialize)]
struct SemgrepResult {
    #[serde(rename = "check_id")]
    check_id: String,
    path: String,
    start: SemgrepPos,
    end: SemgrepPos,
    extra: SemgrepExtra,
}

#[derive(Debug, Deserialize)]
struct SemgrepPos {
    line: u32,
}

#[derive(Debug, Deserialize)]
struct SemgrepExtra {
    message: String,
    severity: String,
    #[serde(default)]
    metadata: SemgrepMeta,
}

#[derive(Debug, Deserialize, Default)]
struct SemgrepMeta {
    #[serde(default)]
    impact: String,
    #[serde(default)]
    category: String,
}

pub fn run(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running semgrep (Solana rules) on {:?}", source_dir);

    // Run semgrep with the official Solana ruleset plus any local rules
    // Registry: https://semgrep.dev/p/solana
    let output = Command::new("semgrep")
        .args([
            "--config", "p/solana",
            "--json",
            "--quiet",
            "--no-git-ignore",
            source_dir.to_str().unwrap_or("."),
        ])
        .output()
        .context("Failed to execute semgrep (is it installed? `pip install semgrep`)")?;

    let stdout = String::from_utf8_lossy(&output.stdout);
    if stdout.trim().is_empty() {
        tracing::info!("semgrep: no output");
        return Ok(vec![]);
    }

    let parsed: SemgrepOutput = match serde_json::from_str(&stdout) {
        Ok(p) => p,
        Err(e) => {
            tracing::warn!("semgrep JSON parse failed: {} — snippet: {}", e, &stdout[..stdout.len().min(300)]);
            return Ok(vec![]);
        }
    };

    if !parsed.errors.is_empty() {
        tracing::warn!("semgrep reported {} parse errors", parsed.errors.len());
    }

    let findings = parsed.results.iter().enumerate().map(|(idx, r)| {
        let severity = map_severity(&r.extra.severity, &r.extra.metadata.impact);
        let category = if r.extra.metadata.category.is_empty() {
            derive_category(&r.check_id)
        } else {
            r.extra.metadata.category.clone()
        };

        Finding {
            id: format!("{}-semgrep-{}", target_id, idx),
            target_id: target_id.to_string(),
            bounty_type: Some("solana-program".to_string()),
            category: format!("semgrep:{}", category),
            severity,
            tools: vec!["semgrep".to_string()],
            location: Location {
                file: r.path.clone(),
                line_start: r.start.line,
                line_end: r.end.line,
            },
            description: format!("[{}] {}", r.check_id, r.extra.message),
        }
    }).collect::<Vec<_>>();

    tracing::info!("semgrep found {} results", findings.len());
    Ok(findings)
}

fn map_severity(semgrep_sev: &str, impact: &str) -> Severity {
    // Prefer impact metadata when present (more precise)
    match impact.to_lowercase().as_str() {
        "critical" => return Severity::Critical,
        "high" => return Severity::High,
        "medium" => return Severity::Medium,
        "low" => return Severity::Low,
        _ => {}
    }
    match semgrep_sev.to_uppercase().as_str() {
        "ERROR" => Severity::High,
        "WARNING" => Severity::Medium,
        "INFO" => Severity::Low,
        _ => Severity::Info,
    }
}

fn derive_category(check_id: &str) -> String {
    // check_id looks like "solana.security.missing-signer-check" — take the last segment
    check_id.rsplit('.').next().unwrap_or(check_id).to_string()
}
