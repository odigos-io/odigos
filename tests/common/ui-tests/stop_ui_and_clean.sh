#!/bin/bash

# Ensure the script fails if any command fails
set -e

echo "Killing Odigos UI process"
kill "$(cat odigos-ui.pid)"
rm odigos-ui.pid
rm ../../odigos-ui.log