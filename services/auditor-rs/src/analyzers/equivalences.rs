// Category equivalence map between Aderyn and Slither.
pub fn are_equivalent(aderyn_cat: &str, slither_cat: &str) -> bool {
    match (aderyn_cat, slither_cat) {
        ("reentrancy-state-change", "reentrancy-eth") => true,
        ("unchecked-call", "unchecked-lowlevel") => true,
        ("missing-zero-check", "missing-zero-address-validation") => true,
        _ => false,
    }
}
