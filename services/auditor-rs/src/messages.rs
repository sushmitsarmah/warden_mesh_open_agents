use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "lowercase")]
pub enum TargetKind { Onchain, Github, Immunefi }

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct Target {
    pub id: String,
    pub kind: TargetKind,
    #[serde(rename = "chainId", skip_serializing_if = "Option::is_none")]
    pub chain_id: Option<u64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub address: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub repo: Option<String>,
    #[serde(rename = "commitSha", skip_serializing_if = "Option::is_none")]
    pub commit_sha: Option<String>,
    #[serde(rename = "sourceUrl", skip_serializing_if = "Option::is_none")]
    pub source_url: Option<String>,
    #[serde(rename = "discoveredAt")]
    pub discovered_at: DateTime<Utc>,
    pub priority: f64,
    #[serde(rename = "tvlUsd", skip_serializing_if = "Option::is_none")]
    pub tvl_usd: Option<f64>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
#[serde(rename_all = "lowercase")]
pub enum Severity { Info, Low, Medium, High, Critical }

#[derive(Debug, Serialize, Deserialize, Clone, Default)]
pub struct Location {
    pub file: String,
    #[serde(rename = "lineStart")]
    pub line_start: u32,
    #[serde(rename = "lineEnd")]
    pub line_end: u32,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct Finding {
    pub id: String,
    #[serde(rename = "targetId")]
    pub target_id: String,
    pub category: String,
    pub severity: Severity,
    pub tools: Vec<String>,
    #[serde(default)]
    pub location: Location,
    #[serde(default)]
    pub description: String,
}
