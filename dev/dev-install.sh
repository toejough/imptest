#!/usr/bin/env sh

echo "Checking for fish shell..."
if ! command -v fish &> /dev/null; then
    echo "...not found"
    echo "Checking for brew..."
    if ! command -v brew &> /dev/null; then
        echo "...not found"
        echo "Installing brew"
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    fi

    echo "Checking for brew..."
    if ! command -v brew &> /dev/null; then
        echo "brew installation failed somehow. cannot continue to install dev tooling."
        exit 1
    fi
    echo "...found"

    echo "...installing fish"
    brew install fish

    echo "Checking for fish shell..."
    if ! command -v fish &> /dev/null; then
        echo "...not found"
        echo "fish installation failed somehow. cannot continue to install dev tooling."
        exit 1
    fi
fi
echo "...found"

# from https://stackoverflow.com/a/246128
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

$SCRIPT_DIR/dev-install.fish
