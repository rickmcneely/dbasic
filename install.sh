#!/bin/bash
# DBasic Installer
# Installs the DBasic transpiler and adds it to PATH

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "  DBasic Installer"
echo "  BASIC-to-Go Transpiler"
echo "=========================================="
echo

# Check for Go installation
echo -n "Checking for Go installation... "
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    echo -e "${GREEN}Found $GO_VERSION${NC}"
else
    echo -e "${RED}NOT FOUND${NC}"
    echo
    echo "Go is required to build DBasic."
    echo "Please install Go from: https://go.dev/dl/"
    echo
    exit 1
fi

# Check Go version (need at least 1.18 for generics)
GO_MAJOR=$(go version | awk '{print $3}' | sed 's/go//' | cut -d. -f1)
GO_MINOR=$(go version | awk '{print $3}' | sed 's/go//' | cut -d. -f2)

if [ "$GO_MAJOR" -lt 1 ] || ([ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 18 ]); then
    echo -e "${RED}Error: Go 1.18 or later is required (found go$GO_MAJOR.$GO_MINOR)${NC}"
    exit 1
fi

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Build DBasic
echo -n "Building DBasic... "
if go build -o dbasic ./cmd/dbasic; then
    echo -e "${GREEN}OK${NC}"
else
    echo -e "${RED}FAILED${NC}"
    exit 1
fi

# Determine install location
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    # Try user's local bin
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

# Install the binary
echo -n "Installing to $INSTALL_DIR... "
if cp "$SCRIPT_DIR/dbasic" "$INSTALL_DIR/dbasic"; then
    chmod +x "$INSTALL_DIR/dbasic"
    echo -e "${GREEN}OK${NC}"
else
    echo -e "${RED}FAILED${NC}"
    echo "Try running with sudo: sudo ./install.sh"
    exit 1
fi

# Check if INSTALL_DIR is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo
    echo -e "${YELLOW}Warning: $INSTALL_DIR is not in your PATH${NC}"
    echo
    echo "Add the following line to your ~/.bashrc or ~/.profile:"
    echo
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    echo
    echo "Then run: source ~/.bashrc"
    echo
fi

# Verify installation
echo -n "Verifying installation... "
if "$INSTALL_DIR/dbasic" version &> /dev/null || "$INSTALL_DIR/dbasic" help &> /dev/null; then
    echo -e "${GREEN}OK${NC}"
else
    # Try running it to see if it works at all
    if "$INSTALL_DIR/dbasic" 2>&1 | grep -q "Usage\|DBasic\|dbasic"; then
        echo -e "${GREEN}OK${NC}"
    else
        echo -e "${YELLOW}Could not verify (may still work)${NC}"
    fi
fi

echo
echo -e "${GREEN}=========================================="
echo "  DBasic installed successfully!"
echo "==========================================${NC}"
echo
echo "Usage:"
echo "  dbasic check <file.dbas>   - Check syntax"
echo "  dbasic emit <file.dbas>    - Generate Go code"
echo "  dbasic build <file.dbas>   - Build executable"
echo "  dbasic run <file.dbas>     - Build and run"
echo
echo "Example:"
echo "  dbasic run examples/hello.dbas"
echo
