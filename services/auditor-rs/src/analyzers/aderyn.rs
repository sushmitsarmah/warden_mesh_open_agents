use std::path::Path;
use std::process::Command;
use std::fs;
use crate::messages::{Finding, Severity, Location};
use anyhow::{Context, Result};

pub fn run_aderyn(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running aderyn on {:?}", source_dir);

    let report_path = source_dir.join("report.md");

    // Remove old report if it exists
    let _ = fs::remove_file(&report_path);

    let output = Command::new("aderyn")
        .arg(".")
        .current_dir(source_dir)
        .output()
        .context("Failed to execute aderyn")?;

    // Aderyn sometimes crashes but still generates a report
    if !report_path.exists() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        tracing::warn!("Aderyn did not generate report.md: {}", stderr);
        return Ok(vec![]);
    }

    let report = fs::read_to_string(&report_path)
        .context("Failed to read aderyn report")?;

    let findings = parse_aderyn_markdown(&report, target_id);

    tracing::info!("Aderyn found {} issues", findings.len());
    Ok(findings)
}

fn parse_aderyn_markdown(markdown: &str, target_id: &str) -> Vec<Finding> {
    let mut findings = Vec::new();
    let lines: Vec<&str> = markdown.lines().collect();
    let mut idx = 0;

    for (line_num, line) in lines.iter().enumerate() {
        // Look for issue headers like "## H-1:", "## M-2:", "## L-3:"
        if line.starts_with("## ") {
            let severity = if line.contains("## H-") {
                Severity::High
            } else if line.contains("## M-") {
                Severity::Medium
            } else if line.contains("## L-") {
                Severity::Low
            } else if line.contains("## C-") {
                Severity::Critical
            } else {
                continue;
            };

            // Extract category from the header (everything after the ID)
            let category = line
                .split(':')
                .nth(1)
                .unwrap_or("unknown")
                .trim()
                .to_string();

            // Try to find location info in the following lines
            let mut location = Location::default();
            let mut description = category.clone();

            // Look ahead for file location (e.g., "- Found in src/...")
            for i in (line_num + 1)..lines.len().min(line_num + 20) {
                let next_line = lines[i];

                if next_line.starts_with("## ") {
                    break; // Hit next section
                }

                if next_line.contains("- Found in ") {
                    // Parse: "- Found in src/demo/VulnerableVault.sol [Line: 11]"
                    if let Some(file_part) = next_line.split("Found in ").nth(1) {
                        let file = file_part
                            .split(" [Line:")
                            .next()
                            .unwrap_or("")
                            .trim()
                            .to_string();

                        let line_start = file_part
                            .split("[Line: ")
                            .nth(1)
                            .and_then(|s| s.split(']').next())
                            .and_then(|s| s.parse().ok())
                            .unwrap_or(0);

                        location = Location {
                            file,
                            line_start,
                            line_end: line_start,
                        };
                    }
                    break;
                }

                // Collect description from paragraphs
                if !next_line.is_empty() && !next_line.starts_with('#') && !next_line.starts_with('-') {
                    description = next_line.trim().to_string();
                }
            }

            findings.push(Finding {
                id: format!("{}-aderyn-{}", target_id, idx),
                target_id: target_id.to_string(),
                category,
                severity,
                tools: vec!["aderyn".to_string()],
                location,
                description,
            });

            idx += 1;
        }
    }

    findings
}
