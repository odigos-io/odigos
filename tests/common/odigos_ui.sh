#!/bin/bash

# Ensure the script fails if any command fails
set -e

scripts_dir="$(cd "$(dirname "$0")" && pwd)"
# The above "$scripts_dir" key is used to identify where the script was called from, to ensure all paths are relative to the script.
# This is useful when the script is called from another location, and the paths are relative to the calling script (for exmaple YAML file).

log_file="$scripts_dir/odigos_ui.log"
pid_file="$scripts_dir/odigos_ui.pid"

function get_process_id() {
  if [ ! -f "$1" ]; then
    # File does not exist
    echo "0"
    return
  fi

  pid=$(cat "$1" 2>/dev/null)
  if ps -p "$pid" > /dev/null 2>&1; then
    # Process is running
    echo "$pid"
  else
    # Process is not running
    echo "0"
  fi
}

function check_process() {
  # Check if the process is running
  if [ "$1" == 0 ]; then
    echo "Odigos UI - ❌ Failed to start"
    cat "$log_file"
    exit 1
  else
    echo "Odigos UI - ✅ Ready"
    cat "$log_file"
  fi
}

function kill_process() {
  # Kill the process
  if [ "$1" != 0 ]; then
    echo "Odigos UI - 💀 Killing process ($1)"
    kill $1
  fi
}

function kill_all() {
  pid=$(get_process_id "$pid_file")
  kill_process $pid
}

function stop() {
  kill_all

  # Cleanup
  rm -f "$log_file"
  rm -f "$pid_file"
}

function start() {
  kill_all
  cd "$scripts_dir/../../frontend/webapp"

  # Install dependencies
  echo "Odigos UI - ⏳ Installing..."
  yarn install > /dev/null 2> "$log_file"

  # Create a production build
  echo "Odigos UI - ⏳ Building..."
  yarn build > /dev/null 2> "$log_file"
  yarn back:build > /dev/null 2> "$log_file"

  # Start the production build
  echo "Odigos UI - ⏳ Starting..."
  yarn back:start > /dev/null 2> "$log_file" &

  sleep 3
  echo $! > "$pid_file"
  pid=$(get_process_id "$pid_file")
  check_process $pid
}

function test() {
  # Run tests on the Frontend
  cd "$scripts_dir/../../frontend/webapp"
  echo "Odigos UI - 👀 Testing with Cypress..."

  set +e # Temporarily disable "exit on error"
  yarn cy
  test_exit_code=$?
  set -e # Re-enable "exit on error"

  if [ $test_exit_code -ne 0 ]; then
    echo "Odigos UI - ❌ Cypress tests failed"
    exit 1
  else
    echo "Odigos UI - ✅ Cypress tests passed"
  fi
}

# This is to allow the script to be used dynamically, we call the function name from the CLI (start/stop/test/etc.)
# This method prevents duplicated code across multiple-scripts
function main() {
  if [ $# -lt 1 ]; then
    echo "❌ Error: Incorrect usage - '$0 <function_name>'"
    exit 1
  fi

  func="$1"
  shift # Shift arguments, so $@ contains only the arguments for the function

  # Check if the function exists and call it (with the remaining arguments)
  if declare -f "$func" > /dev/null; then
    $func "$@"
  else
    echo "❌ Error: Function '$func' not found."
    exit 1
  fi
}

main "$@"
