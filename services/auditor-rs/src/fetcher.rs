use std::collections::HashMap;
use serde::{Deserialize, Serialize};
use anyhow::{Context, bail};

#[derive(Debug, Serialize, Deserialize)]
pub struct SourceBundle {
    pub files: HashMap<String, String>,
    pub abi: String,
    pub compiler_version: String,
}

#[derive(Debug, Deserialize)]
#[serde(untagged)]
enum EtherscanResult {
    Success(Vec<EtherscanContractSource>),
    Error(String),
}

#[derive(Debug, Deserialize)]
struct EtherscanResponse {
    status: String,
    message: String,
    result: EtherscanResult,
}

#[derive(Debug, Deserialize)]
struct EtherscanContractSource {
    #[serde(rename = "SourceCode")]
    source_code: String,
    #[serde(rename = "ABI")]
    abi: String,
    #[serde(rename = "ContractName")]
    contract_name: String,
    #[serde(rename = "CompilerVersion")]
    compiler_version: String,
}

pub async fn fetch_source(chain_id: u64, address: &str) -> anyhow::Result<SourceBundle> {
    let api_key = std::env::var("ETHERSCAN_API_KEY").unwrap_or_else(|_| String::from("YourApiKeyToken"));

    // Select appropriate Etherscan API endpoint based on chain ID
    let base_url = match chain_id {
        1 => "https://api.etherscan.io/api",
        11155111 => "https://api-sepolia.etherscan.io/api", // Sepolia
        _ => bail!("Unsupported chain ID: {}", chain_id),
    };

    let url = format!(
        "{}?module=contract&action=getsourcecode&address={}&apikey={}",
        base_url, address, api_key
    );

    let client = reqwest::Client::new();
    let response = client
        .get(&url)
        .send()
        .await
        .context("Failed to fetch from Etherscan API")?;

    let response_text = response
        .text()
        .await
        .context("Failed to read Etherscan response")?;

    // Try to parse as JSON
    let etherscan_resp: EtherscanResponse = serde_json::from_str(&response_text)
        .with_context(|| format!("Failed to parse Etherscan response. Response body: {}",
            if response_text.len() > 200 { &response_text[..200] } else { &response_text }))?;

    if etherscan_resp.status != "1" {
        let error_msg = match etherscan_resp.result {
            EtherscanResult::Error(ref err) => err.clone(),
            _ => etherscan_resp.message.clone(),
        };
        bail!("Etherscan API error: {}", error_msg);
    }

    let contracts = match etherscan_resp.result {
        EtherscanResult::Success(contracts) => contracts,
        EtherscanResult::Error(err) => bail!("Etherscan API error: {}", err),
    };

    if contracts.is_empty() {
        bail!("No source code found for address {}", address);
    }

    let contract = &contracts[0];

    if contract.source_code.is_empty() {
        bail!("Contract {} is not verified on Etherscan", address);
    }

    let mut files = HashMap::new();

    // Handle multi-file contracts (JSON-encoded source)
    if contract.source_code.starts_with('{') {
        // Try to parse as multi-file JSON structure
        if let Ok(multi_file) = serde_json::from_str::<serde_json::Value>(&contract.source_code) {
            if let Some(sources) = multi_file.get("sources").and_then(|s| s.as_object()) {
                for (filename, file_data) in sources {
                    if let Some(content) = file_data.get("content").and_then(|c| c.as_str()) {
                        files.insert(filename.clone(), content.to_string());
                    }
                }
            }
        }

        // If parsing failed or no files found, treat as single file
        if files.is_empty() {
            files.insert(
                format!("{}.sol", contract.contract_name),
                contract.source_code.clone(),
            );
        }
    } else {
        // Single file contract
        files.insert(
            format!("{}.sol", contract.contract_name),
            contract.source_code.clone(),
        );
    }

    Ok(SourceBundle {
        files,
        abi: contract.abi.clone(),
        compiler_version: contract.compiler_version.clone(),
    })
}
