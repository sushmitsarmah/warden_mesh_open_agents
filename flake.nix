{
  description = "Autonomous Web3 Security Swarm — reproducible dev environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        # Python with slither-analyzer available
        pythonWithSlither = pkgs.python311.withPackages (ps: with ps; [
          slither-analyzer
        ]);
      in
      {
        devShells.default = pkgs.mkShell {
          name = "web3-security-swarm";

          buildInputs = with pkgs; [
            # Go toolchain
            go_1_23
            gopls
            govulncheck

            # Rust toolchain
            rustc
            cargo
            clippy
            rustfmt
            rust-analyzer

            # Foundry (forge, cast, anvil, chisel)
            foundry

            # Node.js (for dashboard frontend if needed)
            nodejs_20
            yarn

            # Python + Slither
            pythonWithSlither

            # Base utilities
            gnumake
            jq
            git
            curl
            wget
            which

            # Go-ethereum tools (includes abigen)
            go-ethereum

            # Nice-to-haves
            direnv
            nix-direnv
          ];

          shellHook = ''
            echo "╔══════════════════════════════════════════════════════════════╗"
            echo "║    Autonomous Web3 Security Swarm — Dev Environment          ║"
            echo "╠══════════════════════════════════════════════════════════════╣"
            echo "║  Go        $(go version | awk '{print $3}')"
            echo "║  Rust      $(rustc --version | awk '{print $2}')"
            echo "║  Foundry   $(forge --version 2>/dev/null || echo 'not found')"
            echo "║  Node      $(node --version)"
            echo "║  Python    $(python3 --version | awk '{print $2}')"
            echo "║  Slither   $(slither --version 2>/dev/null || echo 'not found')"
            echo "╠══════════════════════════════════════════════════════════════╣"
            echo "║  Quick start:                                                ║"
            echo "║    make build                                                ║"
            echo "║    make test                                                 ║"
            echo "╚══════════════════════════════════════════════════════════════╝"

            # Ensure .env exists
            if [ ! -f .env ]; then
              if [ -f .env.example ]; then
                cp .env.example .env
                echo ""
                echo "⚠️  Created .env from .env.example — please fill in your API keys!"
              fi
            fi

            # Auto-install Aderyn if missing (it's not in nixpkgs yet)
            if ! command -v aderyn &> /dev/null; then
              echo ""
              echo "📦 Installing Aderyn via cargo (one-time)..."
              cargo install aderyn
              if command -v aderyn &> /dev/null; then
                echo "✅ Aderyn installed successfully"
              else
                echo "⚠️  Aderyn install failed — some tests will not work"
              fi
            fi

            # Ensure abigen is in PATH (go-ethereum)
            export PATH="$(go env GOPATH)/bin:$PATH"
          '';
        };
      });
}
