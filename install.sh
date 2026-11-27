#!/bin/sh
# AxoProof POSIX Installer (macOS Apple Silicon/Intel, Linux x86_64/ARM64, Android Termux)
# Usage: curl -fsSL https://raw.githubusercontent.com/r17v/axoproof/main/install.sh | sh
set -eu

REPO="r17v/axoproof"
INSTALL_DIR="${AXOPROOF_INSTALL_DIR:-$HOME/.local/bin}"
SKILL_DIR="${AXOPROOF_SKILL_DIR:-$HOME/.agents/skills/axoproof}"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

case "$OS" in
  darwin) PLATFORM="darwin-$ARCH" ;;
  linux*) PLATFORM="linux-$ARCH" ;;
  msys*|mingw*|cygwin*)
    echo "Detected Git Bash / MSYS on Windows. Run this in PowerShell instead:" >&2
    echo "  irm https://raw.githubusercontent.com/$REPO/main/install.ps1 | iex" >&2
    exit 1
    ;;
  *)
    echo "Unsupported OS: $OS (on Windows PowerShell, run: irm https://raw.githubusercontent.com/$REPO/main/install.ps1 | iex)" >&2
    exit 1
    ;;
esac

mkdir -p "$INSTALL_DIR"
BIN_NAME="axoproof-$PLATFORM"
BASE_URL="https://github.com/$REPO/releases/latest/download"
DOWNLOAD_URL="$BASE_URL/$BIN_NAME"
CHECKSUM_URL="$BASE_URL/checksums.txt"
TMP_BIN="$(mktemp)"
TMP_SUM="$(mktemp)"
trap 'rm -f "$TMP_BIN" "$TMP_SUM"' EXIT

verify_sha256() {
  if ! curl -fsSL "$CHECKSUM_URL" -o "$TMP_SUM" 2>/dev/null; then
    return 0
  fi
  EXPECTED="$(awk -v name="$BIN_NAME" '$2 == name { print $1 }' "$TMP_SUM")"
  if [ -z "$EXPECTED" ]; then
    return 0
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL="$(sha256sum "$TMP_BIN" | awk '{ print $1 }')"
  elif command -v shasum >/dev/null 2>&1; then
    ACTUAL="$(shasum -a 256 "$TMP_BIN" | awk '{ print $1 }')"
  else
    return 0
  fi
  if [ "$EXPECTED" != "$ACTUAL" ]; then
    echo "ERROR: SHA-256 checksum mismatch for $BIN_NAME (expected $EXPECTED, got $ACTUAL)" >&2
    exit 1
  fi
  echo "==> Verified SHA-256 checksum ($ACTUAL)"
}

echo "==> Downloading AxoProof ($PLATFORM) from $DOWNLOAD_URL ..."
if curl -fsSL "$DOWNLOAD_URL" -o "$TMP_BIN" 2>/dev/null; then
  verify_sha256
  chmod +x "$TMP_BIN"
  mv "$TMP_BIN" "$INSTALL_DIR/axoproof"
  if [ "$OS" = "darwin" ] && command -v xattr >/dev/null 2>&1; then
    xattr -d com.apple.quarantine "$INSTALL_DIR/axoproof" 2>/dev/null || true
  fi
  echo "==> Installed axoproof binary to $INSTALL_DIR/axoproof"
elif command -v go >/dev/null 2>&1; then
  echo "==> Release binary unavailable; building from source via Go ..."
  GOBIN="$INSTALL_DIR" CGO_ENABLED=0 go install "github.com/$REPO/cmd/axoproof@latest"
  echo "==> Built and installed axoproof to $INSTALL_DIR/axoproof"
else
  echo "ERROR: Could not download binary from $DOWNLOAD_URL and Go is not installed." >&2
  exit 1
fi

# Install 1-door Agent Skill + 6 specialist modules into ~/.agents/skills/axoproof
RAW_BASE="https://raw.githubusercontent.com/$REPO/main/skills/axoproof"
mkdir -p "$SKILL_DIR/modules"
if curl -fsSL "$RAW_BASE/SKILL.md" -o "$SKILL_DIR/SKILL.md" 2>/dev/null; then
  for mod in no-placebo cash-register door-locks live-persistence customer-ready self-repair; do
    curl -fsSL "$RAW_BASE/modules/$mod.md" -o "$SKILL_DIR/modules/$mod.md" 2>/dev/null || true
  done
  echo "==> Installed AxoProof Agent Skill to $SKILL_DIR"
fi

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    if [ "$OS" = "darwin" ]; then
      echo "NOTE: Add $INSTALL_DIR to your macOS zsh PATH:"
      echo "  echo 'export PATH=\"$INSTALL_DIR:\$PATH\"' >> ~/.zshrc && export PATH=\"$INSTALL_DIR:\$PATH\""
    else
      echo "NOTE: Add $INSTALL_DIR to your PATH:"
      echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    fi
    ;;
esac

"$INSTALL_DIR/axoproof" version
