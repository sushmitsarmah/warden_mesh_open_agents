use std::path::Path;
use std::fs;
use crate::messages::{Finding, Severity, Location};
use anyhow::Result;

/// Detects oracle-related vulnerabilities in Solidity contracts:
///
/// 1. Chainlink `latestRoundData()` without staleness checks (missing `updatedAt` validation)
/// 2. Chainlink answer not validated (missing `answer > 0` check)
/// 3. Uniswap V3 `slot0()` used as a price source (spot price — trivially manipulable)
/// 4. Single-block TWAP (observation window too short)
/// 5. `latestAnswer()` used instead of `latestRoundData()` (deprecated, no staleness data)
pub fn run(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running oracle staleness checker on {:?}", source_dir);

    let mut findings = Vec::new();
    let mut idx = 0usize;

    scan_dir(source_dir, source_dir, target_id, &mut findings, &mut idx)?;

    tracing::info!("oracle check: {} findings", findings.len());
    Ok(findings)
}

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
                tracing::warn!("oracle_check: scan_file error on {:?}: {}", path, e);
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

    // ---- Check 1: latestRoundData without staleness guard ----
    // A correct usage must check `updatedAt` within a max-age window.
    if content.contains("latestRoundData") {
        let has_staleness = content.contains("updatedAt")
            || content.contains("maxStaleness")
            || content.contains("staleThreshold")
            || content.contains("MAX_DELAY")
            || content.contains("STALENESS")
            || content.contains("heartbeat");

        let has_round_check = content.contains("answeredInRound")
            || content.contains("roundId");

        if !has_staleness {
            let line = find_line(&content, "latestRoundData");
            findings.push(Finding {
                id: format!("{}-oracle-{}", target_id, idx),
                target_id: target_id.to_string(),
                bounty_type: None,
                category: "oracle:missing-staleness-check".to_string(),
                severity: Severity::High,
                tools: vec!["oracle-check".to_string()],
                location: Location { file: rel.clone(), line_start: line, line_end: line },
                description:
                    "Chainlink `latestRoundData()` called without checking `updatedAt` against a \
                    staleness threshold. If the Chainlink oracle goes offline or is paused, \
                    `updatedAt` will be stale and the protocol will use an outdated price. \
                    Add: `if (block.timestamp - updatedAt > MAX_ORACLE_DELAY) revert StalePrice();`"
                    .to_string(),
            });
            *idx += 1;
        }

        if !has_round_check {
            let line = find_line(&content, "latestRoundData");
            findings.push(Finding {
                id: format!("{}-oracle-{}", target_id, idx),
                target_id: target_id.to_string(),
                bounty_type: None,
                category: "oracle:missing-round-check".to_string(),
                severity: Severity::Medium,
                tools: vec!["oracle-check".to_string()],
                location: Location { file: rel.clone(), line_start: line, line_end: line },
                description:
                    "Chainlink `latestRoundData()` called without validating \
                    `answeredInRound >= roundId`. An incomplete round can return a stale answer \
                    from the previous round. Add: \
                    `if (answeredInRound < roundId) revert IncompleteRound();`"
                    .to_string(),
            });
            *idx += 1;
        }

        // Check 2: answer not validated for positivity
        let has_answer_check = content.contains("answer > 0")
            || content.contains("answer >= 0")
            || content.contains("price > 0")
            || content.contains("require(answer")
            || content.contains("if (answer");

        if !has_answer_check {
            let line = find_line(&content, "latestRoundData");
            findings.push(Finding {
                id: format!("{}-oracle-{}", target_id, idx),
                target_id: target_id.to_string(),
                bounty_type: None,
                category: "oracle:unvalidated-answer".to_string(),
                severity: Severity::High,
                tools: vec!["oracle-check".to_string()],
                location: Location { file: rel.clone(), line_start: line, line_end: line },
                description:
                    "Chainlink `latestRoundData()` answer is not checked for positivity. \
                    A zero or negative answer can crash price calculations (division by zero) \
                    or allow manipulation. Add: `if (answer <= 0) revert InvalidPrice();`"
                    .to_string(),
            });
            *idx += 1;
        }
    }

    // ---- Check 3: latestAnswer() (deprecated Chainlink API) ----
    if content.contains(".latestAnswer()") || content.contains("latestAnswer()") {
        let line = find_line(&content, "latestAnswer");
        findings.push(Finding {
            id: format!("{}-oracle-{}", target_id, idx),
            target_id: target_id.to_string(),
            bounty_type: None,
            category: "oracle:deprecated-latest-answer".to_string(),
            severity: Severity::Medium,
            tools: vec!["oracle-check".to_string()],
            location: Location { file: rel.clone(), line_start: line, line_end: line },
            description:
                "`latestAnswer()` is deprecated by Chainlink and provides no round or staleness \
                data. Use `latestRoundData()` and validate `updatedAt`, `answeredInRound`, \
                and `answer > 0`."
                .to_string(),
        });
        *idx += 1;
    }

    // ---- Check 4: Uniswap V3 slot0() used as price source ----
    if content.contains("slot0()") || content.contains(".slot0(") {
        // slot0 is fine for TWAP observation setup, but using sqrtPriceX96 directly is dangerous
        let uses_sqrt_price = content.contains("sqrtPriceX96")
            || content.contains("sqrtPrice");
        let uses_twap = content.contains("observe(")
            || content.contains("OracleLibrary")
            || content.contains("consult(");

        if uses_sqrt_price && !uses_twap {
            let line = find_line(&content, "slot0");
            findings.push(Finding {
                id: format!("{}-oracle-{}", target_id, idx),
                target_id: target_id.to_string(),
                bounty_type: None,
                category: "oracle:spot-price-from-slot0".to_string(),
                severity: Severity::Critical,
                tools: vec!["oracle-check".to_string()],
                location: Location { file: rel.clone(), line_start: line, line_end: line },
                description:
                    "Uniswap V3 `slot0().sqrtPriceX96` used as a price oracle. This is the \
                    current spot price and is trivially manipulable within a single block using \
                    a flash loan. Use `OracleLibrary.consult()` with a TWAP window of at least \
                    30 minutes instead."
                    .to_string(),
            });
            *idx += 1;
        }
    }

    // ---- Check 5: Uniswap V2 getReserves() used as price source ----
    if content.contains("getReserves()") || content.contains("getReserves(") {
        let line = find_line(&content, "getReserves");
        // Only flag if it looks like price derivation (division of reserves)
        if content.contains("reserve0") && content.contains("reserve1")
            && (content.contains("/ reserve") || content.contains("price"))
        {
            findings.push(Finding {
                id: format!("{}-oracle-{}", target_id, idx),
                target_id: target_id.to_string(),
                bounty_type: None,
                category: "oracle:spot-price-from-reserves".to_string(),
                severity: Severity::Critical,
                tools: vec!["oracle-check".to_string()],
                location: Location { file: rel.clone(), line_start: line, line_end: line },
                description:
                    "Uniswap V2 `getReserves()` used to derive an on-chain price. Reserve-based \
                    spot prices are flash-loan manipulable within a single transaction. \
                    Use a Chainlink price feed or a Uniswap V3 TWAP with a minimum 30-minute \
                    observation window."
                    .to_string(),
            });
            *idx += 1;
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
