use std::collections::HashMap;
use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize)]
pub struct SourceBundle {
    pub files: HashMap<String, String>,
    pub abi: String,
    pub compiler_version: String,
}

pub async fn fetch_source(chain_id: u64, address: &str) -> anyhow::Result<SourceBundle> {
    let _ = (chain_id, address);
    Ok(SourceBundle {
        files: HashMap::new(),
        abi: String::new(),
        compiler_version: String::new(),
    })
}
