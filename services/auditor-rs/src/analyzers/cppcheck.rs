use std::path::Path;
use std::process::Command;
use crate::messages::{Finding, Severity, Location};
use anyhow::{Context, Result};

pub fn run(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running cppcheck on {:?}", source_dir);

    let output = Command::new("cppcheck")
        .args(&[
            "--xml",
            "--enable=all",
            "--suppress=*:*/test*",
            "--suppress=*:*/ci/*",
            "--suppress=*:*/scripts/*",
            source_dir.to_str().unwrap_or("."),
        ])
        .output()
        .context("Failed to execute cppcheck")?;

    if !output.status.success() && output.stdout.is_empty() && output.stderr.is_empty() {
        tracing::warn!("cppcheck produced no output");
        return Ok(vec![]);
    }

    let stderr = String::from_utf8_lossy(&output.stderr);
    let findings = parse_cppcheck_xml(&stderr, target_id);

    tracing::info!("cppcheck found {} issues", findings.len());
    Ok(findings)
}

fn parse_cppcheck_xml(xml: &str, target_id: &str) -> Vec<Finding> {
    let mut findings = Vec::new();
    let mut idx = 0usize;

    for line in xml.lines() {
        let trimmed = line.trim();
        if !trimmed.starts_with("<error ") {
            continue;
        }

        let severity = extract_attr(trimmed, "severity");
        let msg = extract_attr(trimmed, "msg");
        let file = extract_attr(trimmed, "file");
        let line_str = extract_attr(trimmed, "line");

        let severity = match severity.as_str() {
            "error" => Severity::Critical,
            "warning" => Severity::High,
            "style" => Severity::Medium,
            "performance" => Severity::Medium,
            "portability" => Severity::Low,
            "information" => Severity::Info,
            _ => Severity::Info,
        };

        let line_start: u32 = line_str.parse().unwrap_or(0);

        findings.push(Finding {
                id: format!("{}-cppcheck-{}", target_id, idx),
                target_id: target_id.to_string(),
                bounty_type: None,
                category: "static-analysis".to_string(),
                severity,
                tools: vec!["cppcheck".to_string()],
                location: Location {
                    file: file.clone(),
                    line_start,
                    line_end: line_start,
                },
                description: msg,
            });
        idx += 1;
    }

    findings
}

fn extract_attr(s: &str, attr: &str) -> String {
    let pattern = format!("{}=\"", attr);
    if let Some(start) = s.find(&pattern) {
        let val_start = start + pattern.len();
        if let Some(end) = s[val_start..].find('"') {
            return s[val_start..val_start + end].to_string();
        }
    }
    String::new()
}
