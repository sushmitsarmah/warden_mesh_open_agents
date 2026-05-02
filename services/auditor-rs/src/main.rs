mod messages;
mod axl;
mod fetcher;

use auditor::analyzers::{aderyn, slither, merge};
use std::collections::HashMap;
use std::fs;
use std::path::PathBuf;
use tokio::task;

const FOUNDRY_TOML: &str = "[profile.default]\nsrc = \".\"\nout = \"out\"\nlibs = []\n";

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt::init();
    tracing::info!("auditor: starting");

    let mut stream = axl::subscribe_targets().await;
    while let Some(target) = stream.next().await {
        tracing::info!("received target: {}", target.id);

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

        let work_dir = std::env::temp_dir().join(format!("auditor-{}", target.id));
        if let Err(e) = prepare_work_dir(&work_dir, &bundle.files) {
            tracing::error!("failed to prepare work dir for {}: {}", target.id, e);
            continue;
        }

        let dir = work_dir.clone();
        let tid = target.id.clone();
        let findings = task::spawn_blocking(move || {
            let a = aderyn::run_aderyn(&dir, &tid).unwrap_or_else(|e| {
                tracing::warn!("aderyn failed for {}: {}", tid, e);
                vec![]
            });
            let s = slither::run_slither(&dir, &tid).unwrap_or_else(|e| {
                tracing::warn!("slither failed for {}: {}", tid, e);
                vec![]
            });
            merge::merge(a, s)
        })
        .await
        .unwrap_or_default();

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

fn prepare_work_dir(dir: &PathBuf, files: &HashMap<String, String>) -> anyhow::Result<()> {
    fs::create_dir_all(dir)?;
    fs::write(dir.join("foundry.toml"), FOUNDRY_TOML)?;
    for (name, content) in files {
        let dest = dir.join(name);
        if let Some(parent) = dest.parent() {
            fs::create_dir_all(parent)?;
        }
        fs::write(&dest, content)?;
    }
    Ok(())
}
