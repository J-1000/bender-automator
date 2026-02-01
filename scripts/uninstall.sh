#!/bin/bash
set -e

echo "Uninstalling Bender..."

INSTALL_DIR="/usr/local/bin"
LAUNCHAGENT_PATH="$HOME/Library/LaunchAgents/com.bender.daemon.plist"

# Stop daemon if running
if [ -f "$LAUNCHAGENT_PATH" ]; then
    echo "Stopping daemon..."
    launchctl unload "$LAUNCHAGENT_PATH" 2>/dev/null || true
    rm -f "$LAUNCHAGENT_PATH"
fi

# Remove daemon binary
if [ -f "$INSTALL_DIR/benderd" ]; then
    echo "Removing daemon..."
    rm -f "$INSTALL_DIR/benderd"
fi

# Remove CLI
if command -v npm &> /dev/null; then
    echo "Removing CLI..."
    npm unlink @bender/cli 2>/dev/null || true
fi

# Remove socket
rm -f /tmp/bender.sock

echo ""
echo "Bender uninstalled."
echo ""
echo "Config and data directories were preserved:"
echo "  - Config: ~/.config/bender/"
echo "  - Data: ~/.local/share/bender/"
echo "  - Logs: /usr/local/var/log/bender/"
echo ""
echo "To remove all data, run:"
echo "  rm -rf ~/.config/bender ~/.local/share/bender /usr/local/var/log/bender"
echo ""
