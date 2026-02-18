#!/bin/sh
# tb-discover installer — downloads the latest release for your platform.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/escape-velocity-ventures/tb-discover/main/install.sh | sh
#
# Environment variables:
#   TB_DISCOVER_VERSION  — specific version (default: latest)
#   TB_DISCOVER_DIR      — install directory (default: /usr/local/bin)

set -e

REPO="escape-velocity-ventures/tb-discover"
INSTALL_DIR="${TB_DISCOVER_DIR:-/usr/local/bin}"

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64)  ARCH="arm64" ;;
  *)
    echo "Error: unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

case "$OS" in
  linux|darwin) ;;
  *)
    echo "Error: unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# Resolve version
if [ -z "$TB_DISCOVER_VERSION" ]; then
  TB_DISCOVER_VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
  if [ -z "$TB_DISCOVER_VERSION" ]; then
    echo "Error: could not determine latest version" >&2
    exit 1
  fi
fi

VERSION_NUM="${TB_DISCOVER_VERSION#v}"
FILENAME="tb-discover_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${TB_DISCOVER_VERSION}/${FILENAME}"

echo "Downloading tb-discover ${TB_DISCOVER_VERSION} for ${OS}/${ARCH}..."

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

curl -fsSL "$URL" -o "${TMP}/${FILENAME}"
tar -xzf "${TMP}/${FILENAME}" -C "$TMP"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP}/tb-discover" "${INSTALL_DIR}/tb-discover"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "${TMP}/tb-discover" "${INSTALL_DIR}/tb-discover"
fi

chmod +x "${INSTALL_DIR}/tb-discover"
echo "tb-discover ${TB_DISCOVER_VERSION} installed to ${INSTALL_DIR}/tb-discover"
