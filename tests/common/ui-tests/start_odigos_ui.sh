#!/bin/bash

# Ensure the script fails if any command fails
set -e

echo "Running odigos UI setup"
cd ../../../frontend/webapp
odigos ui > ../../odigos-ui.log 2>&1 &

# Capture the process ID
echo $! > odigos-ui.pid

# Check the status of the process
sleep 5
if ps -p $(cat odigos-ui.pid) > /dev/null
then
  echo "Odigos UI started successfully"
else
  echo "Failed to start Odigos UI"
  # I want to print the log file to the console
  cat ../../odigos-ui.log
  exit 1
fi