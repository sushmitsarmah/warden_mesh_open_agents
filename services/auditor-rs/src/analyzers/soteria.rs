use std::path::Path;
use std::process::Command;
use crate::messages::{Finding, Severity, Location};
use serde::Deserialize;
use anyhow::{Context, Result};

#[derive(Debug, Deserialize)]
struct SoteriaReport {
    #[serde(default)]
    vulnerabilities: Vec<SoteriaVuln>,
}

#[derive(Debug, Deserialize)]
struct SoteriaVuln {
    #[serde(rename = "type")]
    vuln_type: String,
    severity: String,
    description: String,
    #[serde(default)]
    file: String,
    #[serde(default)]
    line: u32,
    #[serde(default)]
    function: String,
}

pub fn run(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running Soteria on {:?}", source_dir);

    // Soteria CLI: `sot check --json`
    // Docs: https://www.soteria.dev/
    let output = Command::new("sot")
        .args(["check", "--json"])
        .current_dir(source_dir)
        .output()
        .context("Failed to execute sot (is Soteria installed? `npm i -g @soteria-dev/sot`)")?;

    let stdout = String::from_utf8_lossy(&output.stdout);
    if stdout.trim().is_empty() {
        tracing::info!("soteria: no output");
        return Ok(vec![]);
    }

    let report: SoteriaReport = match serde_json::from_str(&stdout) {
        Ok(r) => r,
        Err(e) => {
            tracing::warn!("soteria JSON parse failed: {} — snippet: {}", e, &stdout[..stdout.len().min(300)]);
            return Ok(vec![]);
        }
    };

    let findings = report.vulnerabilities.iter().enumerate().map(|(idx, v)| {
        let severity = map_severity(&v.severity);
        let desc = if v.function.is_empty() {
            v.description.clone()
        } else {
            format!("[{}] {}: {}", v.vuln_type, v.function, v.description)
        };

        Finding {
            id: format!("{}-soteria-{}", target_id, idx),
            target_id: target_id.to_string(),
            bounty_type: Some("solana-program".to_string()),
            category: format!("soteria:{}", v.vuln_type),
            severity,
            tools: vec!["soteria".to_string()],
            location: Location {
                file: v.file.clone(),
                line_start: v.line,
                line_end: v.line,
            },
            description: desc,
        }
    }).collect::<Vec<_>>();

    tracing::info!("soteria found {} vulnerabilities", findings.len());
    Ok(findings)
}

fn map_severity(s: &str) -> Severity {
    match s.to_lowercase().as_str() {
        "critical" => Severity::Critical,
        "high" => Severity::High,
        "medium" | "moderate" => Severity::Medium,
        "low" | "informational" | "info" => Severity::Low,
        _ => Severity::Medium,
    }
}
