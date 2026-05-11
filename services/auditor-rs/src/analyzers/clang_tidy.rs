use std::path::Path;
use std::process::Command;
use crate::messages::{Finding, Severity, Location};
use anyhow::{Context, Result};
use serde::Deserialize;

#[derive(Debug, Deserialize)]
struct ClangTidyDiagnostic {
    #[serde(default)]
    diagnosticname: String,
    #[serde(default)]
    message: String,
    #[serde(default)]
    filename: String,
    #[serde(default)]
    line: u32,
    #[serde(default)]
    column: u32,
    #[serde(default)]
    level: String,
}

#[derive(Debug, Deserialize)]
struct ClangTidyOutput {
    #[serde(default)]
    diagnostics: Vec<ClangTidyDiagnostic>,
}

pub fn run(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running clang-tidy on {:?}", source_dir);

    // Build first so compile_commands.json exists
    let build_dir = source_dir.join("build");
    std::fs::create_dir_all(&build_dir)?;

    let cmake_result = Command::new("cmake")
        .args(&[
            "-B", build_dir.to_str().unwrap_or("build"),
            "-S", source_dir.to_str().unwrap_or("."),
            "-DCMAKE_EXPORT_COMPILE_COMMANDS=ON",
            "-DCMAKE_C_COMPILER=gcc-14",
            "-DCMAKE_CXX_COMPILER=g++-14",
        ])
        .output();

    match cmake_result {
        Ok(out) if out.status.success() => {
            tracing::info!("cmake configure succeeded");
        }
        _ => {
            tracing::warn!("cmake configure failed, clang-tidy may not find compile_commands.json");
        }
    }

    // Run clang-tidy with -p for build dir
    let tmp_output = source_dir.join("clang-tidy-output.json");
    let output = Command::new("run-clang-tidy")
        .args(&[
            "-p",
            build_dir.to_str().unwrap_or("build"),
            "-export-fixes",
            tmp_output.to_str().unwrap_or("clang-tidy-output.json"),
            "-quiet",
        ])
        .output()
        .context("Failed to execute run-clang-tidy")?;

    let findings = if tmp_output.exists() {
        let content = std::fs::read_to_string(&tmp_output)?;
        parse_clang_tidy_json(&content, target_id)
    } else {
        // Fallback: try parsing stderr
        let stderr = String::from_utf8_lossy(&output.stderr);
        parse_clang_tidy_text(&stderr, target_id)
    };

    tracing::info!("clang-tidy found {} issues", findings.len());
    Ok(findings)
}

fn parse_clang_tidy_json(json: &str, target_id: &str) -> Vec<Finding> {
    let mut findings = Vec::new();

    if let Ok(parsed) = serde_json::from_str::<ClangTidyOutput>(json) {
        for (idx, diag) in parsed.diagnostics.iter().enumerate() {
            let severity = match diag.level.as_str() {
                "critical" => Severity::Critical,
                "fatal" => Severity::Critical,
                "error" => Severity::High,
                "warning" => Severity::Medium,
                _ => Severity::Info,
            };

            findings.push(Finding {
                id: format!("{}-clang-tidy-{}", target_id, idx),
                target_id: target_id.to_string(),
                bounty_type: None,
                category: diag.diagnosticname.clone(),
                severity,
                tools: vec!["clang-tidy".to_string()],
                location: Location {
                    file: diag.filename.clone(),
                    line_start: diag.line,
                    line_end: diag.line,
                },
                description: diag.message.clone(),
            });
        }
    }

    findings
}

fn parse_clang_tidy_text(stderr: &str, target_id: &str) -> Vec<Finding> {
    let mut findings = Vec::new();
    let mut idx = 0usize;

    for line in stderr.lines() {
        // Format: /path/file.c:line:col: warning: message [checkname]
        if !line.contains("warning:") && !line.contains("error:") {
            continue;
        }

        let parts: Vec<&str> = line.splitn(5, ':').collect();
        if parts.len() < 4 {
            continue;
        }

        let file = parts[0].trim().to_string();
        let line_num: u32 = parts[1].trim().parse().unwrap_or(0);
        let level = parts[3].trim().to_string();

        let msg = if parts.len() > 4 { parts[4].trim().to_string() } else { level.clone() };

        let severity = match level.as_str() {
            " error" | "error" => Severity::High,
            " warning" | "warning" => Severity::Medium,
            _ => Severity::Info,
        };

        // Extract check name from [...]
        let category = if let Some(start) = msg.rfind('[') {
            if let Some(end) = msg[start..].find(']') {
                msg[start + 1..start + end].to_string()
            } else {
                "unknown".to_string()
            }
        } else {
            "unknown".to_string()
        };

        findings.push(Finding {
                id: format!("{}-clang-tidy-{}", target_id, idx),
                target_id: target_id.to_string(),
                bounty_type: None,
                category,
                severity,
                tools: vec!["clang-tidy".to_string()],
                location: Location {
                    file: file.clone(),
                    line_start: line_num,
                    line_end: line_num,
                },
                description: msg.clone(),
            });
        idx += 1;
    }

    findings
}
