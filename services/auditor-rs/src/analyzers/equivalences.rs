// Category equivalence map between Aderyn and Slither.
pub fn are_equivalent(aderyn_cat: &str, slither_cat: &str) -> bool {
    // Aderyn categories are in the description, Slither uses check field
    // We need to match based on semantic meaning

    // Check for reentrancy patterns
    if (aderyn_cat.to_lowercase().contains("eth") || aderyn_cat.to_lowercase().contains("protected"))
        && slither_cat == "reentrancy-eth" {
        return true;
    }

    match (aderyn_cat, slither_cat) {
        ("reentrancy-state-change", "reentrancy-eth") => true,
        ("unchecked-call", "unchecked-lowlevel") => true,
        ("missing-zero-check", "missing-zero-address-validation") => true,
        ("Missing checks for `address(0)` when assigning values to address state variables", "missing-zero-check") => true,
        _ => false,
    }
}
