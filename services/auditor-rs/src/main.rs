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
                    analyzers::merge::merge(a, s)
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
