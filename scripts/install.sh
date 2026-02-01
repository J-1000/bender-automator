#!/bin/bash
set -e

echo "Installing Bender..."

INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.config/bender"
DATA_DIR="$HOME/.local/share/bender"
LOG_DIR="/usr/local/var/log/bender"

# Create directories
echo "Creating directories..."
mkdir -p "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"

# Check for Go and build daemon
if command -v go &> /dev/null; then
    echo "Building daemon..."
    cd "$(dirname "$0")/../daemon"
    go build -o "$INSTALL_DIR/benderd" ./cmd/benderd
    echo "Daemon installed to $INSTALL_DIR/benderd"
else
    echo "Go not found. Please install Go 1.22+ or download a pre-built binary."
fi

# Install CLI
if command -v npm &> /dev/null; then
    echo "Installing CLI..."
    cd "$(dirname "$0")/../cli"
    npm install
    npm run build
    npm link
    echo "CLI installed. Run 'bender' to use."
else
    echo "npm not found. Please install Node.js 20+ to use the CLI."
fi

# Copy default config if not exists
if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    echo "Creating default config..."
    cp "$(dirname "$0")/../configs/default.yaml" "$CONFIG_DIR/config.yaml"
fi

echo ""
echo "Bender installed successfully!"
echo ""
echo "Next steps:"
echo "  1. Configure: bender config"
echo "  2. Install LaunchAgent: bender install agent"
echo "  3. Start daemon: bender start"
echo ""
