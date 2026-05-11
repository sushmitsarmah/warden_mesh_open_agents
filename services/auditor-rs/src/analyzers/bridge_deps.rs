use std::path::Path;
use std::fs;
use crate::messages::{Finding, Severity, Location};
use anyhow::Result;

/// Bridge dependency scanner — detects cross-chain bridge protocol integrations
/// in Solidity contracts and flags missing validation patterns.
///
/// Cross-chain bridges are among the highest-risk third-party dependencies in DeFi.
/// This scanner:
/// 1. Identifies which bridge protocols a contract depends on.
/// 2. Flags missing message-origin validation (spoofed sender attacks).
/// 3. Flags missing chain ID checks (replay across chains).
/// 4. Flags missing nonce/uniqueness checks (replay of same message).
/// 5. Flags contracts that trust bridge messages without verifying payload integrity.
pub fn run(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running bridge dependency scanner on {:?}", source_dir);

    let mut findings = Vec::new();
    let mut idx = 0usize;

    scan_dir(source_dir, source_dir, target_id, &mut findings, &mut idx)?;

    tracing::info!("bridge scanner: {} findings", findings.len());
    Ok(findings)
}

// ---- bridge protocol signatures ---- //

struct BridgeSignature {
    name: &'static str,
    /// Strings that indicate this bridge is imported/used
    import_markers: &'static [&'static str],
    /// The function/callback that receives cross-chain messages
    receive_fn: &'static [&'static str],
    /// Required validation that MUST be present near the receive function
    required_checks: &'static [(&'static str, &'static str)], // (pattern, description)
    /// Severity when the bridge is used but validation is missing
    missing_validation_severity: Severity,
}

const BRIDGES: &[BridgeSignature] = &[
    BridgeSignature {
        name: "LayerZero",
        import_markers: &["ILayerZeroEndpoint", "ILayerZeroReceiver", "lzReceive", "LayerZeroEndpoint"],
        receive_fn: &["lzReceive(", "lzReceive ("],
        required_checks: &[
            ("trustedRemote", "Missing `trustedRemote` check — caller can spoof source chain/address"),
            ("_srcChainId", "Source chain ID not validated — messages from unintended chains accepted"),
        ],
        missing_validation_severity: Severity::Critical,
    },
    BridgeSignature {
        name: "Wormhole",
        import_markers: &["IWormhole", "IWormholeRelayer", "IWormholeReceiver", "wormhole.parseAndVerifyVM"],
        receive_fn: &["receiveWormholeMessages(", "processWormholeMessage("],
        required_checks: &[
            ("verifyVM", "Missing `verifyVM` call — Wormhole VAA not cryptographically verified"),
            ("emitterAddress", "Emitter address not checked — attacker can send messages from arbitrary emitters"),
            ("emitterChainId", "Emitter chain ID not validated — cross-chain replay possible"),
            ("processedMessages", "No message replay protection — same VAA can be processed multiple times"),
        ],
        missing_validation_severity: Severity::Critical,
    },
    BridgeSignature {
        name: "CCTP (Circle)",
        import_markers: &["IMessageTransmitter", "ITokenMessenger", "CCTP", "MessageTransmitter"],
        receive_fn: &["receiveMessage(", "handleReceiveMessage("],
        required_checks: &[
            ("sourceDomain", "Source domain not validated — messages from wrong chain accepted"),
            ("sender", "Message sender not verified against trusted remote"),
        ],
        missing_validation_severity: Severity::High,
    },
    BridgeSignature {
        name: "Axelar",
        import_markers: &["IAxelarGateway", "IAxelarExecutable", "AxelarExecutable", "IAxelarGasService"],
        receive_fn: &["execute(", "_execute(", "executeWithToken("],
        required_checks: &[
            ("validateContractCall", "Missing `gateway.validateContractCall` — message authenticity not verified"),
            ("sourceChain", "Source chain not validated against allowlist"),
            ("sourceAddress", "Source address not validated against trusted remote"),
        ],
        missing_validation_severity: Severity::Critical,
    },
    BridgeSignature {
        name: "Stargate",
        import_markers: &["IStargateRouter", "IStargateReceiver", "sgReceive", "Stargate"],
        receive_fn: &["sgReceive("],
        required_checks: &[
            ("stargateRouter", "Missing check that `msg.sender` is the Stargate router"),
            ("_srcChainId", "Source chain ID not validated"),
        ],
        missing_validation_severity: Severity::High,
    },
    BridgeSignature {
        name: "Across",
        import_markers: &["SpokePoolInterface", "IAcrossSpokePool", "AcrossMessageHandler", "handleAcrossMessage"],
        receive_fn: &["handleAcrossMessage("],
        required_checks: &[
            ("spokePool", "Missing check that `msg.sender` is the Across SpokePool"),
        ],
        missing_validation_severity: Severity::High,
    },
    BridgeSignature {
        name: "Hyperlane",
        import_markers: &["IMailbox", "IMessageRecipient", "IInterchainSecurityModule", "handle(uint32"],
        receive_fn: &["handle(uint32", "handle (uint32"],
        required_checks: &[
            ("mailbox", "Missing check that `msg.sender` is the Hyperlane mailbox"),
            ("ISM", "No Interchain Security Module configured — messages not cryptographically verified"),
            ("origin", "Origin domain not validated"),
        ],
        missing_validation_severity: Severity::Critical,
    },
];

fn scan_dir(
    root: &Path,
    dir: &Path,
    target_id: &str,
    findings: &mut Vec<Finding>,
    idx: &mut usize,
) -> Result<()> {
    let Ok(entries) = fs::read_dir(dir) else { return Ok(()) };
    for entry in entries.flatten() {
        let path = entry.path();
        if path.is_dir() {
            let name = path.file_name().and_then(|n| n.to_str()).unwrap_or("");
            if matches!(name, "node_modules" | ".git" | "out" | "cache" | "lib") {
                continue;
            }
            scan_dir(root, &path, target_id, findings, idx)?;
        } else if path.extension().and_then(|e| e.to_str()) == Some("sol") {
            if let Err(e) = scan_file(root, &path, target_id, findings, idx) {
                tracing::warn!("bridge_deps: scan_file error on {:?}: {}", path, e);
            }
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
    let content = fs::read_to_string(path)?;
    let rel = path.strip_prefix(root).unwrap_or(path).to_string_lossy().to_string();

    if rel.contains("test") || rel.contains("Test") || rel.contains("mock") || rel.contains("Mock") {
        return Ok(());
    }

    for bridge in BRIDGES {
        // Check if this file uses the bridge
        let uses_bridge = bridge.import_markers.iter().any(|m| content.contains(m));
        if !uses_bridge {
            continue;
        }

        // Report the bridge dependency itself as an informational finding
        let import_line = bridge.import_markers.iter()
            .find_map(|m| if content.contains(m) { Some(find_line(&content, m)) } else { None })
            .unwrap_or(0);

        findings.push(Finding {
            id: format!("{}-bridge-dep-{}", target_id, idx),
            target_id: target_id.to_string(),
            bounty_type: None,
            category: format!("bridge:dependency:{}", bridge.name.to_lowercase().replace(' ', "-")),
            severity: Severity::Info,
            tools: vec!["bridge-scanner".to_string()],
            location: Location { file: rel.clone(), line_start: import_line, line_end: import_line },
            description: format!(
                "Contract depends on the {} bridge. Bridge contracts are high-value attack \
                targets — ensure all message-receive paths validate origin, chain ID, and \
                message authenticity before trusting payload contents.",
                bridge.name,
            ),
        });
        *idx += 1;

        // Check receive functions for missing validation
        for receive_fn in bridge.receive_fn {
            if !content.contains(receive_fn) {
                continue;
            }

            let fn_line = find_line(&content, receive_fn);

            // Extract the function body for localised checks
            let fn_body = extract_fn_body(&content, receive_fn);

            for (check_pattern, check_desc) in bridge.required_checks {
                let present = content.contains(check_pattern)
                    || fn_body.as_deref().map(|b| b.contains(check_pattern)).unwrap_or(false);

                if !present {
                    findings.push(Finding {
                        id: format!("{}-bridge-val-{}", target_id, idx),
                        target_id: target_id.to_string(),
                        bounty_type: None,
                        category: format!("bridge:missing-validation:{}", bridge.name.to_lowercase().replace(' ', "-")),
                        severity: bridge.missing_validation_severity.clone(),
                        tools: vec!["bridge-scanner".to_string()],
                        location: Location { file: rel.clone(), line_start: fn_line, line_end: fn_line },
                        description: format!(
                            "[{}] `{}`: {}. Unvalidated bridge messages allow an attacker to \
                            forge cross-chain calls and execute arbitrary actions with the \
                            contract's authority.",
                            bridge.name, receive_fn.trim_end_matches('('), check_desc,
                        ),
                    });
                    *idx += 1;
                }
            }
        }
    }

    Ok(())
}

fn find_line(content: &str, needle: &str) -> u32 {
    for (i, line) in content.lines().enumerate() {
        if line.contains(needle) {
            return (i + 1) as u32;
        }
    }
    0
}

/// Extracts roughly the body of the first function containing `fn_marker`.
fn extract_fn_body(content: &str, fn_marker: &str) -> Option<String> {
    let start = content.find(fn_marker)?;
    let after = &content[start..];
    let brace_start = after.find('{')?;
    let rest = &after[brace_start..];

    let mut depth = 0i32;
    let mut end = rest.len();
    for (i, c) in rest.char_indices() {
        match c {
            '{' => depth += 1,
            '}' => {
                depth -= 1;
                if depth == 0 {
                    end = i + 1;
                    break;
                }
            }
            _ => {}
        }
    }
    Some(rest[..end].to_string())
}
