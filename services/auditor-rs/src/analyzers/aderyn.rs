use std::path::Path;
use crate::messages::Finding;

pub fn run_aderyn(source_dir: &Path, target_id: &str) -> anyhow::Result<Vec<Finding>> {
    let _ = (source_dir, target_id);
    Ok(vec![])
}
