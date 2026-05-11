use std::path::Path;
use std::process::Command;
use crate::messages::{Finding, Severity, Location};
use serde::Deserialize;
use anyhow::{Context, Result};

#[derive(Debug, Deserialize)]
struct AuditReport {
    vulnerabilities: AuditVulns,
    #[serde(default)]
    warnings: std::collections::HashMap<String, Vec<serde_json::Value>>,
}

#[derive(Debug, Deserialize)]
struct AuditVulns {
    list: Vec<AuditEntry>,
}

#[derive(Debug, Deserialize)]
struct AuditEntry {
    advisory: Advisory,
    package: PackageInfo,
}

#[derive(Debug, Deserialize)]
struct Advisory {
    id: String,
    title: String,
    severity: Option<String>,
    description: String,
    #[serde(default)]
    categories: Vec<String>,
}

#[derive(Debug, Deserialize)]
struct PackageInfo {
    name: String,
    version: String,
}

pub fn run(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running cargo audit on {:?}", source_dir);

    let output = Command::new("cargo")
        .args(["audit", "--json", "-q"])
        .current_dir(source_dir)
        .output()
        .context("Failed to execute cargo audit")?;

    let stdout = String::from_utf8_lossy(&output.stdout);
    if stdout.trim().is_empty() {
        tracing::info!("cargo audit: no output (no vulnerabilities or no Cargo.lock)");
        return Ok(vec![]);
    }

    let report: AuditReport = match serde_json::from_str(&stdout) {
        Ok(r) => r,
        Err(e) => {
            tracing::warn!("cargo audit JSON parse failed: {} — raw: {}", e, &stdout[..stdout.len().min(200)]);
            return Ok(vec![]);
        }
    };

    let mut findings = Vec::new();
    for (idx, entry) in report.vulnerabilities.list.iter().enumerate() {
        let severity = match entry.advisory.severity.as_deref().unwrap_or("") {
            "critical" => Severity::Critical,
            "high" => Severity::High,
            "medium" => Severity::Medium,
            "low" => Severity::Low,
            _ => Severity::Medium,
        };

        let description = format!(
            "[{}] {} v{}: {} — {}",
            entry.advisory.id,
            entry.package.name,
            entry.package.version,
            entry.advisory.title,
            entry.advisory.description,
        );

        findings.push(Finding {
            id: format!("{}-cargo-audit-{}", target_id, idx),
            target_id: target_id.to_string(),
            bounty_type: Some("solana-program".to_string()),
            category: "dependency-vulnerability".to_string(),
            severity,
            tools: vec!["cargo-audit".to_string()],
            location: Location {
                file: "Cargo.lock".to_string(),
                line_start: 0,
                line_end: 0,
            },
            description,
        });
    }

    tracing::info!("cargo audit found {} vulnerabilities", findings.len());
    Ok(findings)
}
