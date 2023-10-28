#!/bin/bash

# Check if Odigos is installed
odigos_version=$(odigos version 2>&1)

if [[ $odigos_version == *"command not found"* ]]; then
    echo "Error: Odigos is not installed. Aborting."
    exit 1
fi

# Odigos is installed, continue with the UI command
odigos ui
