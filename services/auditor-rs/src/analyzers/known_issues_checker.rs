use crate::messages::Finding;

/// Known issue trackers per scope.md:75-95.
/// In a real system these would be fetched from URLs at runtime.
/// For now we maintain a known-pattern list that gets updated by the scope monitor.
const KNOWN_ISSUE_PATTERNS: &[&str] = &[
    // Known issues listed in scope.md:93-95
    "Forged TPU TowerSync vote can create false forward confirmations",
    "fd_rdisp 128 requested-writable transaction counter wrap crashes replay",
    "Mid-slot invalid chained Merkle root child FEC can crash Firedancer replay scheduler",
];

/// Fingerprint a finding to check if it matches a known issue.
/// Uses (file + pattern_key) as the fingerprint.
fn fingerprint(f: &Finding) -> String {
    format!("{}:{}:{}", f.location.file, f.category, f.description.chars().take(80).collect::<String>())
}

/// Filter out findings that match known issues.
/// Private known issues are not eligible per assets.md:106.
pub fn filter(findings: &mut Vec<Finding>) {
    let initial = findings.len();

    findings.retain(|f| {
        let fp = fingerprint(f);
        let is_known = KNOWN_ISSUE_PATTERNS.iter().any(|p| fp.contains(p));

        if is_known {
            tracing::info!(
                "[KNOWN_ISSUE] suppressing finding {} at {}:{} - matches known issue",
                f.id,
                f.location.file,
                f.location.line_start,
            );
        }

        !is_known
    });

    let suppressed = initial - findings.len();
    if suppressed > 0 {
        tracing::info!("known_issues: suppressed {} of {} findings", suppressed, initial);
    }
}
