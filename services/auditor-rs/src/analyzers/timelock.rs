use std::path::Path;
use std::fs;
use crate::messages::{Finding, Severity, Location};
use anyhow::Result;

/// Detects admin/privileged functions in Solidity that are not protected by a timelock.
///
/// A timelock (e.g., OpenZeppelin TimelockController) forces a mandatory delay
/// between proposing and executing sensitive admin operations, giving users time
/// to exit before changes take effect. Missing timelocks on fund-affecting admin
/// functions are a critical DeFi pre-launch risk (checklist item 4).
pub fn run(source_dir: &Path, target_id: &str) -> Result<Vec<Finding>> {
    tracing::info!("Running timelock detector on {:?}", source_dir);

    let mut findings = Vec::new();
    let mut idx = 0usize;

    // Detect whether the project uses any timelock at all
    let has_timelock = project_has_timelock(source_dir);

    scan_sol_dir(source_dir, source_dir, target_id, has_timelock, &mut findings, &mut idx)?;

    tracing::info!("timelock detector: {} findings", findings.len());
    Ok(findings)
}

fn scan_sol_dir(
    root: &Path,
    dir: &Path,
    target_id: &str,
    has_timelock: bool,
    findings: &mut Vec<Finding>,
    idx: &mut usize,
) -> Result<()> {
    let entries = match fs::read_dir(dir) {
        Ok(e) => e,
        Err(_) => return Ok(()),
    };
    for entry in entries.flatten() {
        let path = entry.path();
        if path.is_dir() {
            let name = path.file_name().and_then(|n| n.to_str()).unwrap_or("");
            if matches!(name, "node_modules" | ".git" | "out" | "cache" | "lib") {
                continue;
            }
            scan_sol_dir(root, &path, target_id, has_timelock, findings, idx)?;
        } else if path.extension().and_then(|e| e.to_str()) == Some("sol") {
            scan_file(root, &path, target_id, has_timelock, findings, idx)?;
        }
    }
    Ok(())
}

fn scan_file(
    root: &Path,
    path: &Path,
    target_id: &str,
    has_timelock: bool,
    findings: &mut Vec<Finding>,
    idx: &mut usize,
) -> Result<()> {
    let content = match fs::read_to_string(path) {
        Ok(c) => c,
        Err(_) => return Ok(()),
    };
    let rel = path.strip_prefix(root).unwrap_or(path).to_string_lossy().to_string();

    // Skip test files and mock contracts
    if rel.contains("test") || rel.contains("Test") || rel.contains("mock") || rel.contains("Mock") {
        return Ok(());
    }

    // Skip if this file IS a timelock implementation
    if content.contains("TimelockController") && content.contains("schedule") && content.contains("execute") {
        return Ok(());
    }

    let lines: Vec<&str> = content.lines().collect();
    let mut in_function = false;
    let mut fn_name = String::new();
    let mut fn_start_line = 0u32;
    let mut fn_modifiers: Vec<String> = Vec::new();
    let mut brace_depth = 0i32;
    let mut fn_body = String::new();

    for (i, &line) in lines.iter().enumerate() {
        let line_no = (i + 1) as u32;
        let trimmed = line.trim();

        if trimmed.starts_with("//") || trimmed.starts_with("*") {
            continue;
        }

        // Detect function start
        if !in_function && (trimmed.starts_with("function ") || trimmed.contains(" function ")) {
            fn_name = extract_fn_name(trimmed);
            fn_start_line = line_no;
            fn_modifiers = extract_modifiers(trimmed);
            fn_body = trimmed.to_string();
            brace_depth = trimmed.chars().filter(|&c| c == '{').count() as i32
                - trimmed.chars().filter(|&c| c == '}').count() as i32;
            if brace_depth > 0 {
                in_function = true;
            }
            continue;
        }

        if in_function {
            fn_body.push(' ');
            fn_body.push_str(trimmed);
            brace_depth += trimmed.chars().filter(|&c| c == '{').count() as i32;
            brace_depth -= trimmed.chars().filter(|&c| c == '}').count() as i32;

            if brace_depth <= 0 {
                // Function complete — analyse it
                check_function(
                    &fn_name,
                    fn_start_line,
                    &fn_modifiers,
                    &fn_body,
                    &rel,
                    target_id,
                    has_timelock,
                    findings,
                    idx,
                );
                in_function = false;
                fn_name.clear();
                fn_modifiers.clear();
                fn_body.clear();
                brace_depth = 0;
            }
        }
    }

    Ok(())
}

fn check_function(
    name: &str,
    line: u32,
    modifiers: &[String],
    body: &str,
    file: &str,
    target_id: &str,
    has_timelock: bool,
    findings: &mut Vec<Finding>,
    idx: &mut usize,
) {
    // Is this function privileged?
    let is_privileged = modifiers.iter().any(|m| is_access_modifier(m))
        || is_admin_fn_name(name);

    if !is_privileged {
        return;
    }

    // Does the function touch funds or critical parameters?
    let is_fund_affecting = body_affects_funds(body);
    let is_param_changing = body_changes_params(body);

    if !is_fund_affecting && !is_param_changing {
        return;
    }

    // Is there a timelock guard in the body or does the project use one?
    let has_local_timelock = body.contains("timelock") || body.contains("Timelock")
        || body.contains("delay") || body.contains("schedule")
        || body.contains("TimelockController");

    if has_local_timelock || has_timelock {
        return; // Protected
    }

    let impact = if is_fund_affecting { "fund-affecting" } else { "parameter-changing" };
    let mods = modifiers.join(", ");

    findings.push(Finding {
        id: format!("{}-timelock-{}", target_id, idx),
        target_id: target_id.to_string(),
        bounty_type: None,
        category: "missing-timelock".to_string(),
        severity: if is_fund_affecting { Severity::High } else { Severity::Medium },
        tools: vec!["timelock-detector".to_string()],
        location: Location { file: file.to_string(), line_start: line, line_end: line },
        description: format!(
            "Privileged {} function `{}` (modifiers: [{}]) has no timelock protection. \
            Admin can immediately change critical protocol parameters or move funds without \
            giving users time to exit. Add OpenZeppelin TimelockController with a 24-48h delay.",
            impact, name, mods,
        ),
    });
    *idx += 1;
}

// ---- helpers ---- //

fn is_access_modifier(m: &str) -> bool {
    matches!(
        m,
        "onlyOwner" | "onlyAdmin" | "onlyRole" | "onlyGovernance"
            | "onlyOperator" | "onlyController" | "onlyManager"
            | "onlyGuardian" | "restricted" | "onlyMultisig"
    ) || m.starts_with("onlyRole")
}

fn is_admin_fn_name(name: &str) -> bool {
    let lower = name.to_lowercase();
    lower.starts_with("set")
        || lower.starts_with("update")
        || lower.starts_with("change")
        || lower.starts_with("upgrade")
        || lower.starts_with("pause")
        || lower.starts_with("unpause")
        || lower.starts_with("migrate")
        || lower.starts_with("drain")
        || lower.starts_with("rescue")
        || lower.starts_with("sweep")
        || lower.starts_with("withdraw")
        || lower.contains("emergency")
}

fn body_affects_funds(body: &str) -> bool {
    body.contains("transfer(")
        || body.contains("transferFrom(")
        || body.contains("safeTransfer")
        || body.contains("call{value")
        || body.contains(".call(")
        || body.contains("send(")
        || body.contains("withdraw")
        || body.contains("drain")
        || body.contains("rescue")
        || body.contains("sweep")
}

fn body_changes_params(body: &str) -> bool {
    // Assignments to storage variables (simplified heuristic)
    body.contains(" = ")
        && (body.contains("Fee") || body.contains("fee")
            || body.contains("Rate") || body.contains("rate")
            || body.contains("Limit") || body.contains("limit")
            || body.contains("Oracle") || body.contains("oracle")
            || body.contains("Address") || body.contains("address"))
}

fn project_has_timelock(root: &Path) -> bool {
    // Quick check — does any .sol file reference TimelockController?
    fn check_dir(dir: &Path) -> bool {
        let Ok(entries) = fs::read_dir(dir) else { return false };
        for entry in entries.flatten() {
            let path = entry.path();
            if path.is_dir() {
                let name = path.file_name().and_then(|n| n.to_str()).unwrap_or("");
                if !matches!(name, "node_modules" | ".git" | "out" | "cache") && check_dir(&path) {
                    return true;
                }
            } else if path.extension().and_then(|e| e.to_str()) == Some("sol") {
                if let Ok(content) = fs::read_to_string(&path) {
                    if content.contains("TimelockController") || content.contains("ITimelock") {
                        return true;
                    }
                }
            }
        }
        false
    }
    check_dir(root)
}

fn extract_fn_name(line: &str) -> String {
    // "function foo(..." → "foo"
    if let Some(start) = line.find("function ") {
        let rest = &line[start + 9..];
        let end = rest.find('(').unwrap_or(rest.len());
        return rest[..end].trim().to_string();
    }
    String::new()
}

fn extract_modifiers(line: &str) -> Vec<String> {
    // Everything between closing ')' of params and '{' or 'returns'
    let Some(close_paren) = line.rfind(')') else { return vec![] };
    let after = &line[close_paren + 1..];
    let end = after.find('{').unwrap_or(after.len());
    let modifier_region = &after[..end];

    // Strip 'returns (...)' clause
    let modifier_region = if let Some(ret) = modifier_region.find("returns") {
        &modifier_region[..ret]
    } else {
        modifier_region
    };

    modifier_region
        .split_whitespace()
        .filter(|w| !matches!(*w, "public" | "private" | "internal" | "external" | "view" | "pure" | "virtual" | "override" | "payable"))
        .filter(|w| !w.is_empty() && !w.starts_with('(') && !w.starts_with(')'))
        .map(|w| w.trim_matches(|c: char| !c.is_alphanumeric() && c != '_').to_string())
        .filter(|w| !w.is_empty())
        .collect()
}
