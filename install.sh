#!/bin/sh
set -e

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
    x86_64)          ARCH="amd64" ;;
    aarch64|arm64)   ARCH="arm64" ;;
    *) echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

VERSION=$(curl -fsSL https://api.github.com/repos/laeioun/cue/releases/latest \
    | grep '"tag_name"' | cut -d'"' -f4)

INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

curl -fsSL \
    "https://github.com/laeioun/cue/releases/download/${VERSION}/cue_${OS}_${ARCH}" \
    -o "$INSTALL_DIR/cue"
chmod +x "$INSTALL_DIR/cue"

# Add to PATH for this session so cue install can run immediately.
export PATH="$PATH:$INSTALL_DIR"

# Add to PATH permanently if not already there.
if ! grep -q '.local/bin' "$HOME/.zshrc" 2>/dev/null && \
   ! grep -q '.local/bin' "$HOME/.bashrc" 2>/dev/null; then
    RCFILE="${HOME}/.zshrc"
    [ -f "$HOME/.bashrc" ] && RCFILE="${HOME}/.bashrc"
    echo 'export PATH="$PATH:$HOME/.local/bin"' >> "$RCFILE"
fi

cue install

echo ""
echo "cue ${VERSION} installed"
echo "  Restart your shell or run: source ~/.zshrc"
