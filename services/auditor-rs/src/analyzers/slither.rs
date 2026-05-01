use std::path::Path;
use std::process::Command;
use crate::messages::{Finding, Severity, Location};
use serde::Deserialize;
use anyhow::{Context, Result};

#[derive(Debug, Deserialize)]
struct SlitherOutput {
    results: SlitherResults,
}

#[derive(Debug, Deserialize)]
struct SlitherResults {
    detectors: Vec<SlitherDetector>,
}

#[derive(Debug, Deserialize)]
struct SlitherDetector {
    check: String,
    impact: String,
    description: String,
    #[serde(default)]
    elements: Vec<SlitherElement>,
}

#[derive(Debug, Deserialize)]
struct SlitherElement {
    #[serde(default)]
    source_mapping: Option<SourceMapping>,
}

#[derive(Debug, Deserialize)]
struct SourceMapping {
    #[serde(default)]
    filename_short: String,
    #[serde(default)]
    lines: Vec<u32>,
}

pub fn run_slither(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running slither on {:?}", source_dir);

    let output = Command::new("slither")
        .arg(".")
        .arg("--json")
        .arg("-")
        .current_dir(source_dir)
        .output()
        .context("Failed to execute slither")?;

    if !output.status.success() && output.stdout.is_empty() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        tracing::warn!("Slither failed: {}", stderr);
        return Ok(vec![]);
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    let parsed: SlitherOutput = serde_json::from_str(&stdout)
        .context("Failed to parse slither JSON output")?;

    let mut findings = Vec::new();
    for (idx, detector) in parsed.results.detectors.iter().enumerate() {
        let severity = match detector.impact.to_lowercase().as_str() {
            "critical" => Severity::Critical,
            "high" => Severity::High,
            "medium" => Severity::Medium,
            "low" => Severity::Low,
            _ => Severity::Info,
        };

        let location = detector.elements.first()
            .and_then(|e| e.source_mapping.as_ref())
            .map(|sm| Location {
                file: sm.filename_short.clone(),
                line_start: sm.lines.first().copied().unwrap_or(0),
                line_end: sm.lines.last().copied().unwrap_or(0),
            })
            .unwrap_or_default();

        findings.push(Finding {
            id: format!("{}-slither-{}", target_id, idx),
            target_id: target_id.to_string(),
            category: detector.check.clone(),
            severity,
            tools: vec!["slither".to_string()],
            location,
            description: detector.description.clone(),
        });
    }

    tracing::info!("Slither found {} issues", findings.len());
    Ok(findings)
}
