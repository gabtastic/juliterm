#!/bin/sh
set -e

REPO="cihann/juliterm"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)      echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64)  ARCH="arm64" ;;
  *)              echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Get latest version
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$LATEST" ]; then
  echo "Failed to fetch latest release." >&2
  exit 1
fi

BINARY="juli-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${BINARY}"

echo "Downloading juli ${LATEST} for ${OS}/${ARCH}..."
TMP=$(mktemp)
curl -fsSL "$URL" -o "$TMP"
chmod +x "$TMP"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP" "${INSTALL_DIR}/juli"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "$TMP" "${INSTALL_DIR}/juli"
fi

echo "juli ${LATEST} installed to ${INSTALL_DIR}/juli"
