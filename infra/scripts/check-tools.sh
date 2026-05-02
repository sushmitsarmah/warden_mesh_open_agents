#!/usr/bin/env bash
# infra/scripts/check-tools.sh
# Checks that every external tool required by the swarm is installed.
# For tools that can be installed automatically (aderyn), prompts to install.
# For system-level tools (go, rust, forge, etc.), warns and exits.

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

MISSING_REQUIRED=()
MISSING_OPTIONAL=()
AUTO_INSTALLABLE=()

# Check a command exists
check_cmd() {
    command -v "$1" >/dev/null 2>&1
}

# ── Required system-level tools (cannot auto-install) ───────────────

echo "Checking system tools..."

for tool in go rustc cargo forge node python3 make jq git curl; do
    if check_cmd "$tool"; then
        echo -e "  ${GREEN}✓${NC} $tool"
    else
        echo -e "  ${RED}✗${NC} $tool — NOT FOUND"
        MISSING_REQUIRED+=("$tool")
    fi
done

# ── Optional: Aderyn (can auto-install via cargo) ───────────────────

echo ""
echo "Checking analyzers..."

if check_cmd aderyn; then
    echo -e "  ${GREEN}✓${NC} aderyn"
else
    echo -e "  ${YELLOW}!${NC} aderyn — not found"
    AUTO_INSTALLABLE+=("aderyn")
fi

if check_cmd slither; then
    echo -e "  ${GREEN}✓${NC} slither"
else
    echo -e "  ${YELLOW}!${NC} slither — not found (try: pip install slither-analyzer)"
    MISSING_OPTIONAL+=("slither")
fi

# ── Handle failures ─────────────────────────────────────────────────

if [ ${#MISSING_REQUIRED[@]} -gt 0 ]; then
    echo ""
    echo -e "${RED}ERROR: Missing required system tools:${NC}"
    for tool in "${MISSING_REQUIRED[@]}"; do
        echo "  - $tool"
    done
    echo ""
    echo "These must be installed via your system package manager."
    echo "If you are using Nix, run:  nix develop"
    echo "Otherwise, install manually (see README.md)."
    exit 1
fi

# ── Auto-install aderyn if missing ──────────────────────────────────

if [ ${#AUTO_INSTALLABLE[@]} -gt 0 ]; then
    echo ""
    for tool in "${AUTO_INSTALLABLE[@]}"; do
        if [ "$tool" == "aderyn" ]; then
            echo "Aderyn can be installed via cargo. Install now? (y/n)"
            read -r REPLY < /dev/tty
            if [[ "$REPLY" =~ ^[Yy]$ ]]; then
                echo "Installing aderyn via cargo..."
                cargo install aderyn
                if check_cmd aderyn; then
                    echo -e "  ${GREEN}✓${NC} aderyn installed successfully"
                else
                    echo -e "  ${RED}✗${NC} aderyn installation failed"
                    exit 1
                fi
            else
                echo "Skipped. Aderyn is required for 'make test-analyzers' to work."
                # Don't fail build — aderyn is only needed for auditor tests
            fi
        fi
    done
fi

# ── Summary ─────────────────────────────────────────────────────────

echo ""
echo -e "${GREEN}All required tools present.${NC}"
if [ ${#MISSING_OPTIONAL[@]} -gt 0 ]; then
    echo -e "${YELLOW}Optional tools missing:${NC}"
    for tool in "${MISSING_OPTIONAL[@]}"; do
        echo "  - $tool"
    done
fi
