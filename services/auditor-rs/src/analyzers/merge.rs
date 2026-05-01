use crate::messages::{Finding, Severity};

pub fn merge(aderyn: Vec<Finding>, slither: Vec<Finding>) -> Vec<Finding> {
    let mut merged = Vec::new();
    let mut slither_matched = vec![false; slither.len()];

    for a in &aderyn {
        let mut found = false;
        for (i, s) in slither.iter().enumerate() {
            if slither_matched[i] {
                continue;
            }
            if crate::analyzers::equivalences::are_equivalent(&a.category, &s.category) {
                // Take max severity, union tools
                let mut f = a.clone();
                f.tools = vec!["aderyn".to_string(), "slither".to_string()];
                f.severity = max_severity(&a.severity, &s.severity);
                merged.push(f);
                slither_matched[i] = true;
                found = true;
                break;
            }
        }
        if !found && is_high_or_critical(&a.severity) {
            let mut f = a.clone();
            f.tools = vec!["aderyn".to_string()];
            merged.push(f);
        }
    }

    for (i, s) in slither.iter().enumerate() {
        if !slither_matched[i] && is_high_or_critical(&s.severity) {
            let mut f = s.clone();
            f.tools = vec!["slither".to_string()];
            merged.push(f);
        }
    }

    merged
}

fn is_high_or_critical(sev: &Severity) -> bool {
    matches!(sev, Severity::High | Severity::Critical)
}

fn max_severity(a: &Severity, b: &Severity) -> Severity {
    use Severity::*;
    match (a, b) {
        (Critical, _) | (_, Critical) => Critical,
        (High, _) | (_, High) => High,
        (Medium, _) | (_, Medium) => Medium,
        (Low, _) | (_, Low) => Low,
        _ => Info,
    }
}
