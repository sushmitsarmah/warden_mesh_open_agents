use auditor::messages::{Target, Finding};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use tokio::sync::Mutex;
use tokio::time::{interval, Duration};

#[derive(Debug, Serialize, Deserialize)]
struct Message {
    topic: String,
    payload: serde_json::Value,
}

pub struct AxlClient {
    api_url: String,
    peer_keys: Vec<String>,
    http_client: reqwest::Client,
}

impl AxlClient {
    pub fn new(peer_keys: Vec<String>) -> Self {
        let api_url = std::env::var("AXL_API_URL")
            .unwrap_or_else(|_| "http://127.0.0.1:9002".to_string());

        tracing::info!("AXL client initialized: api_url={}, peers={}", api_url, peer_keys.len());

        Self {
            api_url,
            peer_keys,
            http_client: reqwest::Client::new(),
        }
    }

    pub async fn publish(&self, topic: &str, payload: serde_json::Value) -> anyhow::Result<()> {
        let msg = Message {
            topic: topic.to_string(),
            payload,
        };

        let msg_bytes = serde_json::to_vec(&msg)?;

        // Broadcast to all peers
        for peer_key in &self.peer_keys {
            if let Err(e) = self.send_to_peer(peer_key, &msg_bytes).await {
                tracing::warn!("Failed to send to peer {}: {}", peer_key, e);
                // Continue to other peers
            }
        }

        tracing::debug!("Published message to topic: {}", topic);
        Ok(())
    }

    async fn send_to_peer(&self, peer_key: &str, data: &[u8]) -> anyhow::Result<()> {
        let resp = self
            .http_client
            .post(format!("{}/send", self.api_url))
            .header("X-Destination-Peer-Id", peer_key)
            .header("Content-Type", "application/octet-stream")
            .body(data.to_vec())
            .send()
            .await?;

        if !resp.status().is_success() {
            let body = resp.text().await?;
            anyhow::bail!("Send failed: {}", body);
        }

        Ok(())
    }

    pub async fn recv(&self) -> anyhow::Result<Option<Message>> {
        let resp = self
            .http_client
            .get(format!("{}/recv", self.api_url))
            .send()
            .await?;

        if resp.status() != reqwest::StatusCode::OK {
            return Ok(None); // No messages
        }

        let data = resp.bytes().await?;
        let msg: Message = serde_json::from_slice(&data)?;
        Ok(Some(msg))
    }

    pub async fn health(&self) -> anyhow::Result<()> {
        let resp = self
            .http_client
            .get(format!("{}/topology", self.api_url))
            .send()
            .await?;

        if !resp.status().is_success() {
            anyhow::bail!("Health check failed");
        }

        Ok(())
    }
}

pub struct TargetStream {
    client: Arc<AxlClient>,
}

impl TargetStream {
    pub async fn next(&mut self) -> Option<Target> {
        // Poll for messages with exponential backoff
        let mut backoff = Duration::from_millis(100);

        loop {
            match self.client.recv().await {
                Ok(Some(msg)) if msg.topic == "targets/discovered" => {
                    match serde_json::from_value::<Target>(msg.payload) {
                        Ok(target) => return Some(target),
                        Err(e) => {
                            tracing::warn!("Failed to parse target: {}", e);
                            continue;
                        }
                    }
                }
                Ok(Some(_)) => {
                    // Message for different topic, continue polling
                    continue;
                }
                Ok(None) => {
                    // No messages, wait and retry
                    tokio::time::sleep(backoff).await;
                    backoff = (backoff * 2).min(Duration::from_secs(5));
                }
                Err(e) => {
                    tracing::error!("Error receiving from AXL: {}", e);
                    tokio::time::sleep(Duration::from_secs(1)).await;
                }
            }
        }
    }
}

pub async fn subscribe_targets() -> TargetStream {
    let peer_keys = load_peer_keys_from_env();
    let client = Arc::new(AxlClient::new(peer_keys));
    TargetStream { client }
}

pub async fn publish_finding(finding: Finding) -> anyhow::Result<()> {
    let peer_keys = load_peer_keys_from_env();
    let client = AxlClient::new(peer_keys);

    let payload = serde_json::to_value(&finding)?;
    client.publish("analysis/findings", payload).await?;

    Ok(())
}

fn load_peer_keys_from_env() -> Vec<String> {
    let raw = std::env::var("AXL_PEER_KEYS").unwrap_or_default();
    if raw.is_empty() {
        return Vec::new();
    }

    raw.split(&[',', '\n', ' '])
        .map(|s| s.trim())
        .filter(|s| !s.is_empty())
        .map(|s| s.to_string())
        .collect()
}
