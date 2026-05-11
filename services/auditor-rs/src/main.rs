mod messages;
mod axl;
mod fetcher;

use auditor::analyzers;
use std::fs;
use std::path::PathBuf;
use tokio::task;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt::init();
    tracing::info!("auditor: starting");

    let mut stream = axl::subscribe_targets().await;
    while let Some(target) = stream.next().await {
        tracing::info!("received target: {} (bounty={:?})", target.id, target.bounty_type);

        let work_dir = std::env::temp_dir().join(format!("auditor-{}", target.id));

        let bounty_type = target.bounty_type.clone().unwrap_or_default();
        let findings = match bounty_type.as_str() {
            "firedancer" => {
                // For C / firedancer: clone repo and run C analyzers
                let repo = target.repo.clone().unwrap_or_else(|| "firedancer-io/firedancer".to_string());
                let branch = target.commit_sha.clone().unwrap_or_else(|| "v1.0".to_string());
                run_firedancer_analysis(&work_dir, &repo, &branch, &target.id).await
            }
            "solana-program" => {
                // Rust-based Solana programs: clone repo and run Solana analyzers
                let repo = target.repo.clone().unwrap_or_default();
                let branch = target.commit_sha.clone().unwrap_or_else(|| "main".to_string());
                run_solana_analysis(&work_dir, &repo, &branch, &target.id).await
            }
            _ => {
                // Default: EVM / Solidity analysis
                let chain_id = match target.chain_id {
                    Some(id) => id,
                    None => {
                        tracing::info!("skipping non-onchain target {}", target.id);
                        continue;
                    }
                };
                let address = match target.address {
                    Some(a) => a,
                    None => {
                        tracing::warn!("target {} has no address, skipping", target.id);
                        continue;
                    }
                };

                let bundle = match fetcher::fetch_source(chain_id, &address).await {
                    Ok(b) => b,
                    Err(e) => {
                        tracing::warn!("fetch_source failed for {}: {}", target.id, e);
                        continue;
                    }
                };

                if let Err(e) = prepare_work_dir(&work_dir, &bundle.files) {
                    tracing::error!("failed to prepare work dir for {}: {}", target.id, e);
                    continue;
                }

                let dir = work_dir.clone();
                let tid = target.id.clone();
                task::spawn_blocking(move || {
                    let a = analyzers::aderyn::run_aderyn(&dir, &tid).unwrap_or_else(|e| {
                        tracing::warn!("aderyn failed for {}: {}", tid, e);
                        vec![]
                    });
                    let s = analyzers::slither::run_slither(&dir, &tid).unwrap_or_else(|e| {
                        tracing::warn!("slither failed for {}: {}", tid, e);
                        vec![]
                    });
                    let mut findings = analyzers::merge::merge(a, s);

                    // Timelock detector — privileged functions without timelock
                    match analyzers::timelock::run(&dir, &tid) {
                        Ok(f) => findings.extend(f),
                        Err(e) => tracing::warn!("timelock detector failed: {}", e),
                    }

                    // Oracle staleness checker — Chainlink/TWAP misuse
                    match analyzers::oracle_check::run(&dir, &tid) {
                        Ok(f) => findings.extend(f),
                        Err(e) => tracing::warn!("oracle check failed: {}", e),
                    }

                    // Bridge dependency scanner — 7 protocols, validation checks
                    match analyzers::bridge_deps::run(&dir, &tid) {
                        Ok(f) => findings.extend(f),
                        Err(e) => tracing::warn!("bridge scanner failed: {}", e),
                    }

                    findings
                })
                .await
                .unwrap_or_default()
            }
        };

        tracing::info!("{} findings for target {}", findings.len(), target.id);

        for finding in findings {
            if let Err(e) = axl::publish_finding(finding).await {
                tracing::error!("publish_finding failed: {}", e);
            }
        }

        let _ = fs::remove_dir_all(&work_dir);
    }
    Ok(())
}

async fn run_firedancer_analysis(
    work_dir: &PathBuf,
    repo: &str,
    branch: &str,
    target_id: &str,
) -> Vec<auditor::messages::Finding> {
    tracing::info!("running firedancer analysis: repo={}, branch={}", repo, branch);

    let dir = work_dir.clone();
    let tid = target_id.to_string();
    let r = repo.to_string();
    let b = branch.to_string();

    task::spawn_blocking(move || {
        // Clone the repo at the given branch/tag
        let clone_result = std::process::Command::new("git")
            .args(&[
                "clone",
                "--branch", &b,
                "--depth", "1",
                &format!("https://github.com/{}.git", r),
                dir.to_str().unwrap_or("/tmp/firedancer"),
            ])
            .output();

        let source_dir = match clone_result {
            Ok(out) if out.status.success() => dir.clone(),
            _ => {
                // If clone fails, try an existing local path
                tracing::warn!("git clone failed, trying local path");
                PathBuf::from("/tmp/firedancer")
            }
        };

        let mut findings = Vec::new();

        // Run cppcheck
        match analyzers::cppcheck::run(&source_dir, &tid) {
            Ok(f) => findings.extend(f),
            Err(e) => tracing::warn!("cppcheck failed: {}", e),
        }

        // Run clang-tidy
        match analyzers::clang_tidy::run(&source_dir, &tid) {
            Ok(f) => findings.extend(f),
            Err(e) => tracing::warn!("clang-tidy failed: {}", e),
        }

        // Run fuzz harnesses
        match analyzers::fuzz_harness::run(&source_dir, &tid) {
            Ok(f) => findings.extend(f),
            Err(e) => tracing::warn!("fuzz harness failed: {}", e),
        }

        // Filter known issues
        analyzers::known_issues_checker::filter(&mut findings);

        tracing::info!("firedancer analysis complete: {} findings", findings.len());
        findings
    })
    .await
    .unwrap_or_default()
}

async fn run_solana_analysis(
    work_dir: &PathBuf,
    repo: &str,
    branch: &str,
    target_id: &str,
) -> Vec<auditor::messages::Finding> {
    tracing::info!("running solana analysis: repo={}, branch={}", repo, branch);

    if repo.is_empty() {
        tracing::warn!("solana analysis: no repo specified for target {}", target_id);
        return vec![];
    }

    let dir = work_dir.clone();
    let tid = target_id.to_string();
    let r = repo.to_string();
    let b = branch.to_string();

    task::spawn_blocking(move || {
        let clone_result = std::process::Command::new("git")
            .args(&[
                "clone",
                "--branch", &b,
                "--depth", "1",
                &format!("https://github.com/{}.git", r),
                dir.to_str().unwrap_or("/tmp/solana-program"),
            ])
            .output();

        let source_dir = match clone_result {
            Ok(out) if out.status.success() => dir.clone(),
            Ok(out) => {
                tracing::warn!("git clone failed: {}", String::from_utf8_lossy(&out.stderr));
                PathBuf::from("/tmp/solana-program")
            }
            Err(e) => {
                tracing::warn!("git clone error: {}", e);
                PathBuf::from("/tmp/solana-program")
            }
        };

        let mut findings = Vec::new();

        // 1. Dependency vulnerability scan (RustSec advisory database)
        match analyzers::cargo_audit::run(&source_dir, &tid) {
            Ok(f) => findings.extend(f),
            Err(e) => tracing::warn!("cargo audit failed: {}", e),
        }

        // 2. License, duplicate-dep, and advisory policy checks
        match analyzers::cargo_deny::run(&source_dir, &tid) {
            Ok(f) => findings.extend(f),
            Err(e) => tracing::warn!("cargo deny failed: {}", e),
        }

        // 3. Unsafe code detection in crate and dependencies
        match analyzers::cargo_geiger::run(&source_dir, &tid) {
            Ok(f) => findings.extend(f),
            Err(e) => tracing::warn!("cargo geiger failed: {}", e),
        }

        // 4. Solana-specific Clippy lints (arithmetic, unwrap, indexing)
        match analyzers::clippy_solana::run(&source_dir, &tid) {
            Ok(f) => findings.extend(f),
            Err(e) => tracing::warn!("clippy (solana) failed: {}", e),
        }

        // 5. Regex pattern scan (14 Solana vuln patterns, no compilation needed)
        match analyzers::solana_patterns::run(&source_dir, &tid) {
            Ok(f) => findings.extend(f),
            Err(e) => tracing::warn!("solana pattern scan failed: {}", e),
        }

        // 6. Semgrep with p/solana ruleset (semantic AST-based patterns)
        match analyzers::semgrep_solana::run(&source_dir, &tid) {
            Ok(f) => findings.extend(f),
            Err(e) => tracing::warn!("semgrep (solana) failed: {}", e),
        }

        // 7. Soteria static analysis (Solana-specific semantic checks)
        match analyzers::soteria::run(&source_dir, &tid) {
            Ok(f) => findings.extend(f),
            Err(e) => tracing::warn!("soteria failed: {}", e),
        }

        // 8. Anchor IDL constraint checker (mutable unconstrained accounts)
        match analyzers::anchor_idl::run(&source_dir, &tid) {
            Ok(f) => findings.extend(f),
            Err(e) => tracing::warn!("anchor idl check failed: {}", e),
        }

        // 9. Trident fuzzer (Anchor programs only — generates + runs fuzz harnesses)
        match analyzers::trident_fuzz::run(&source_dir, &tid) {
            Ok(f) => findings.extend(f),
            Err(e) => tracing::warn!("trident fuzz failed: {}", e),
        }

        tracing::info!("solana analysis complete: {} findings", findings.len());
        findings
    })
    .await
    .unwrap_or_default()
}

fn prepare_work_dir(dir: &PathBuf, files: &std::collections::HashMap<String, String>) -> anyhow::Result<()> {
    fs::create_dir_all(dir)?;
    fs::write(dir.join("foundry.toml"), "[profile.default]\nsrc = \".\"\nout = \"out\"\nlibs = []\n")?;
    for (name, content) in files {
        let dest = dir.join(name);
        if let Some(parent) = dest.parent() {
            fs::create_dir_all(parent)?;
        }
        fs::write(&dest, content)?;
    }
    Ok(())
}
