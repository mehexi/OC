#!/bin/sh
set -eu

REPO_OWNER="${REPO_OWNER:-mehexi}"
REPO_NAME="${REPO_NAME:-OC}"
BIN_NAME="oc"
VERSION="${1:-latest}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
	x86_64|amd64) ARCH="amd64" ;;
	aarch64|arm64) ARCH="arm64" ;;
	*) echo "unsupported arch: $ARCH"; exit 1 ;;
esac

case "$OS" in
	linux|darwin) ;;
	*) echo "unsupported OS: $OS"; exit 1 ;;
esac

if [ "$VERSION" = "latest" ]; then
	DOWNLOAD_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest/download/${BIN_NAME}_${OS}_${ARCH}.tar.gz"
else
	DOWNLOAD_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}/${BIN_NAME}_${OS}_${ARCH}.tar.gz"
fi

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

if command -v curl >/dev/null 2>&1; then
	curl -fsSL "$DOWNLOAD_URL" -o "$TMPDIR/release.tar.gz"
elif command -v wget >/dev/null 2>&1; then
	wget -q "$DOWNLOAD_URL" -O "$TMPDIR/release.tar.gz"
else
	echo "need curl or wget"; exit 1
fi

tar xzf "$TMPDIR/release.tar.gz" -C "$TMPDIR"

if [ -f "$TMPDIR/$BIN_NAME" ]; then
	BIN="$TMPDIR/$BIN_NAME"
else
	echo "binary not found in archive"; exit 1
fi

if [ -w /usr/local/bin ]; then
	INSTALL_DIR="/usr/local/bin"
elif [ -w "$HOME/.local/bin" ]; then
	INSTALL_DIR="$HOME/.local/bin"
	mkdir -p "$INSTALL_DIR"
else
	INSTALL_DIR="$HOME/.local/bin"
	mkdir -p "$INSTALL_DIR"
fi

mv "$BIN" "$INSTALL_DIR/$BIN_NAME"
chmod +x "$INSTALL_DIR/$BIN_NAME"

echo "Installed $BIN_NAME to $INSTALL_DIR/$BIN_NAME"
echo "Make sure $INSTALL_DIR is in your PATH"

# hint: one-liner
# curl -fsSL https://raw.githubusercontent.com/${REPO_OWNER}/${REPO_NAME}/main/scripts/install.sh | bash
