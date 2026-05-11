use std::path::{Path, PathBuf};
use std::process::Command;
use std::fs;
use crate::messages::{Finding, Severity, Location};
use serde::Deserialize;
use anyhow::{Context, Result};

// ---- Anchor IDL schema (subset) ---- //

#[derive(Debug, Deserialize)]
struct AnchorIdl {
    #[serde(default)]
    instructions: Vec<IdlInstruction>,
    #[serde(default)]
    accounts: Vec<IdlAccountDef>,
    #[serde(default)]
    errors: Vec<IdlError>,
}

#[derive(Debug, Deserialize)]
struct IdlInstruction {
    name: String,
    #[serde(default)]
    accounts: Vec<IdlAccount>,
}

#[derive(Debug, Deserialize)]
struct IdlAccount {
    name: String,
    #[serde(rename = "isMut", default)]
    is_mut: bool,
    #[serde(rename = "isSigner", default)]
    is_signer: bool,
    // Anchor constraints live in the IDL as optional metadata
    #[serde(default)]
    pda: Option<serde_json::Value>,
    #[serde(default)]
    relations: Vec<String>,
}

#[derive(Debug, Deserialize)]
struct IdlAccountDef {
    name: String,
}

#[derive(Debug, Deserialize)]
struct IdlError {
    code: u32,
    name: String,
    #[serde(default)]
    msg: String,
}

pub fn run(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running Anchor IDL constraint checker on {:?}", source_dir);

    if !source_dir.join("Anchor.toml").exists() {
        tracing::info!("anchor-idl: no Anchor.toml, skipping");
        return Ok(vec![]);
    }

    // Build the IDL — writes JSON to target/idl/<program>.json
    let build = Command::new("anchor")
        .args(["build", "--idl", "/dev/null"])  // suppress artifact write, we'll find it
        .current_dir(source_dir)
        .output()
        .context("Failed to run `anchor build` (install with `npm i -g @coral-xyz/anchor-cli`)")?;

    if !build.status.success() {
        tracing::warn!("anchor build failed: {}", String::from_utf8_lossy(&build.stderr));
        return Ok(vec![]);
    }

    // Gather IDL files from target/idl/
    let idl_dir = source_dir.join("target/idl");
    let idl_files = collect_idl_files(&idl_dir);

    if idl_files.is_empty() {
        tracing::info!("anchor-idl: no IDL files found in target/idl/");
        return Ok(vec![]);
    }

    let mut findings = Vec::new();
    for idl_path in &idl_files {
        let raw = fs::read_to_string(idl_path)?;
        let idl: AnchorIdl = match serde_json::from_str(&raw) {
            Ok(i) => i,
            Err(e) => {
                tracing::warn!("anchor-idl: failed to parse {:?}: {}", idl_path, e);
                continue;
            }
        };

        check_idl(&idl, idl_path, target_id, &mut findings);
    }

    tracing::info!("anchor-idl: {} constraint findings", findings.len());
    Ok(findings)
}

fn check_idl(
    idl: &AnchorIdl,
    idl_path: &Path,
    target_id: &str,
    findings: &mut Vec<Finding>,
) {
    let file = idl_path.to_string_lossy().to_string();
    let mut idx = findings.len();

    for instr in &idl.instructions {
        for account in &instr.accounts {
            // Mutable account with no signer requirement and no PDA seeds —
            // attacker can pass any account and the program will write to it.
            if account.is_mut
                && !account.is_signer
                && account.pda.is_none()
                && account.relations.is_empty()
            {
                findings.push(Finding {
                    id: format!("{}-anchor-idl-{}", target_id, idx),
                    target_id: target_id.to_string(),
                    bounty_type: Some("solana-program".to_string()),
                    category: "anchor:unconstrained-mutable-account".to_string(),
                    severity: Severity::High,
                    tools: vec!["anchor-idl".to_string()],
                    location: Location {
                        file: file.clone(),
                        line_start: 0,
                        line_end: 0,
                    },
                    description: format!(
                        "Instruction `{}`: account `{}` is mutable but has no signer constraint, \
                        no PDA derivation, and no `has_one` relation. An attacker can supply an \
                        arbitrary account and the program will write to it.",
                        instr.name, account.name,
                    ),
                });
                idx += 1;
            }

            // Signer account that also has PDA seeds — likely a mistake
            // (PDAs cannot sign; this would always fail at runtime)
            if account.is_signer && account.pda.is_some() {
                findings.push(Finding {
                    id: format!("{}-anchor-idl-{}", target_id, idx),
                    target_id: target_id.to_string(),
                    bounty_type: Some("solana-program".to_string()),
                    category: "anchor:signer-on-pda".to_string(),
                    severity: Severity::Medium,
                    tools: vec!["anchor-idl".to_string()],
                    location: Location {
                        file: file.clone(),
                        line_start: 0,
                        line_end: 0,
                    },
                    description: format!(
                        "Instruction `{}`: account `{}` is marked as both a signer and a PDA. \
                        PDAs cannot sign transactions — this constraint will always fail or be \
                        silently skipped, depending on the Anchor version.",
                        instr.name, account.name,
                    ),
                });
                idx += 1;
            }
        }

        // Instruction with zero accounts — may be an admin function with no access control
        if instr.accounts.is_empty() {
            findings.push(Finding {
                id: format!("{}-anchor-idl-{}", target_id, idx),
                target_id: target_id.to_string(),
                bounty_type: Some("solana-program".to_string()),
                category: "anchor:no-account-constraints".to_string(),
                severity: Severity::Low,
                tools: vec!["anchor-idl".to_string()],
                location: Location {
                    file: file.clone(),
                    line_start: 0,
                    line_end: 0,
                },
                description: format!(
                    "Instruction `{}` has zero accounts in the IDL. \
                    If this instruction has side effects, there may be no access control enforced.",
                    instr.name,
                ),
            });
            idx += 1;
        }
    }
}

fn collect_idl_files(dir: &PathBuf) -> Vec<PathBuf> {
    let mut files = Vec::new();
    if let Ok(entries) = fs::read_dir(dir) {
        for entry in entries.flatten() {
            let path = entry.path();
            if path.extension().and_then(|e| e.to_str()) == Some("json") {
                files.push(path);
            }
        }
    }
    files
}
