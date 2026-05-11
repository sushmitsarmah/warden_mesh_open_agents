use std::path::Path;
use std::fs;
use crate::messages::{Finding, Severity, Location};
use anyhow::Result;

/// Pattern-based detector for common Solana program vulnerabilities.
/// Walks all .rs files and applies regex patterns without needing to compile the program.
pub fn run(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running Solana pattern analysis on {:?}", source_dir);

    let mut findings = Vec::new();
    let mut idx = 0usize;

    scan_dir(source_dir, source_dir, target_id, &mut findings, &mut idx)?;

    tracing::info!("Solana pattern scan found {} issues", findings.len());
    Ok(findings)
}

fn scan_dir(
    root: &Path,
    dir: &Path,
    target_id: &str,
    findings: &mut Vec<Finding>,
    idx: &mut usize,
) -> Result<()> {
    let entries = match fs::read_dir(dir) {
        Ok(e) => e,
        Err(_) => return Ok(()),
    };

    for entry in entries.flatten() {
        let path = entry.path();
        if path.is_dir() {
            let name = path.file_name().and_then(|n| n.to_str()).unwrap_or("");
            // Skip non-Rust directories
            if matches!(name, "target" | ".git" | "node_modules" | "test-fixtures") {
                continue;
            }
            scan_dir(root, &path, target_id, findings, idx)?;
        } else if path.extension().and_then(|e| e.to_str()) == Some("rs") {
            scan_file(root, &path, target_id, findings, idx)?;
        }
    }
    Ok(())
}

fn scan_file(
    root: &Path,
    path: &Path,
    target_id: &str,
    findings: &mut Vec<Finding>,
    idx: &mut usize,
) -> Result<()> {
    let content = match fs::read_to_string(path) {
        Ok(c) => c,
        Err(_) => return Ok(()),
    };

    let rel_path = path.strip_prefix(root).unwrap_or(path).to_string_lossy().to_string();

    for (line_num, line) in content.lines().enumerate() {
        let line_no = (line_num + 1) as u32;
        check_patterns(line, line_no, &rel_path, target_id, findings, idx);
    }

    Ok(())
}

fn check_patterns(
    line: &str,
    line_no: u32,
    file: &str,
    target_id: &str,
    findings: &mut Vec<Finding>,
    idx: &mut usize,
) {
    // Skip comments
    let trimmed = line.trim();
    if trimmed.starts_with("//") || trimmed.starts_with("/*") || trimmed.starts_with("*") {
        return;
    }

    let patterns: &[(&str, &str, Severity, &str)] = &[
        // Missing signer check — account used without is_signer verification
        ("account_info", "missing-signer-check", Severity::High,
         "AccountInfo used without is_signer check — ensure authority accounts require signing"),

        // Arbitrary CPI — invoking a program ID that comes from an account (not hardcoded)
        ("invoke(&Instruction", "arbitrary-cpi", Severity::High,
         "invoke() call detected — verify program_id is validated against a trusted constant, not an attacker-controlled account"),

        // Missing ownership check — reading account data without verifying owner
        (".data.borrow()", "missing-ownership-check", Severity::Medium,
         "Account data borrowed without verifying account.owner — attacker may supply a malicious account"),

        // Unsafe arithmetic — unchecked integer ops that can overflow/underflow
        ("checked_add", "unchecked-arithmetic-ok", Severity::Info,
         "checked_add used — good practice confirmed"),

        // Missing bump seed canonicalization
        ("create_program_address", "missing-bump-canonicalization", Severity::High,
         "create_program_address() without canonical bump — use find_program_address() to prevent bump grinding attacks"),

        // Reentrancy risk via CPI before state update
        ("invoke_signed", "cpi-before-state-update", Severity::Medium,
         "invoke_signed() detected — verify state is updated before external CPI calls to prevent reentrancy"),

        // Missing close account check (lamport drain attack)
        ("close_account", "account-close-check", Severity::Medium,
         "close_account pattern — ensure the destination receives lamports and data is zeroed to prevent revival"),

        // Integer truncation cast
        ("as u8", "integer-truncation", Severity::Medium,
         "Truncating cast to u8 — may overflow silently; use try_from() or checked casts"),
        ("as u16", "integer-truncation", Severity::Medium,
         "Truncating cast to u16 — may overflow silently; use try_from() or checked casts"),
        ("as i32", "integer-truncation", Severity::Low,
         "Cast to i32 — verify no sign loss or truncation; use try_from() for safety"),

        // Unchecked account deserialization (sealevel attack surface)
        ("AccountDeserialize::try_deserialize_unchecked", "unchecked-deserialize", Severity::High,
         "try_deserialize_unchecked bypasses discriminator check — attacker can pass wrong account type"),

        // Solana token program CPI without authority check
        ("spl_token::instruction::transfer", "spl-transfer-authority", Severity::High,
         "SPL token transfer — verify authority is a validated signer, not an attacker-controlled account"),

        // Missing rent exemption check
        ("rent.is_exempt", "rent-exempt-check-ok", Severity::Info,
         "rent.is_exempt() checked — good practice confirmed"),

        // Panic in on-chain code (causes validator crash or DoS for that transaction)
        ("panic!(", "on-chain-panic", Severity::Medium,
         "panic!() in on-chain code causes transaction abort — use Err() returns instead"),
        ("unwrap()", "on-chain-unwrap", Severity::Medium,
         "unwrap() in on-chain code panics on None/Err — use '?' or explicit error handling"),
    ];

    for (needle, category, severity, description) in patterns {
        // Skip "ok" patterns that are just informational confirmations
        if category.ends_with("-ok") {
            continue;
        }
        if line.contains(needle) {
            let loc = Location {
                file: file.to_string(),
                line_start: line_no,
                line_end: line_no,
            };

            let finding = Finding {
                id: format!("{}-solpat-{}", target_id, idx),
                target_id: target_id.to_string(),
                bounty_type: Some("solana-program".to_string()),
                category: format!("solana:{}", category),
                severity: severity.clone(),
                tools: vec!["solana-patterns".to_string()],
                location: loc,
                description: format!("{} (line: {})", description, line.trim()),
            };

            findings.push(finding);
            *idx += 1;
        }
    }
}
