#!/bin/bash

# Ensure the script fails if any command fails
set -e

scripts_dir="$(cd "$(dirname "$0")" && pwd)"
# The above "$scripts_dir" key is used to identify where the script was called from, to ensure all paths are relative to the script.
# This is useful when the script is called from another location, and the paths are relative to the calling script (for exmaple YAML file).

log_filename="$scripts_dir/odigos_ui.log"
back_pid_filename="$scripts_dir/ui_backend.pid"
front_pid_filename="$scripts_dir/ui_frontend.pid"

function cleanup() {
  rm -f "$front_pid_filename"
  rm -f "$back_pid_filename"
  rm -f "$log_filename"
}

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
    echo "Odigos UI - ❌ $2 failed to start"
    cat "$3"
    exit 1
  else
    echo "Odigos UI - ✅ $2 is ready"
  fi
}

function kill_process() {
  # Kill the process
  if [ "$1" != 0 ]; then
    echo "Odigos UI - 💀 Killing $2 process ($1)"
    kill $1
  fi
}

function kill_all() {
  # Kill processes if they are still running
  front_pid=$(get_process_id "$front_pid_filename")
  kill_process $front_pid "Frontend"
  back_pid=$(get_process_id "$back_pid_filename")
  kill_process $back_pid "Backend"
}

function stop() {
  kill_all
  cleanup
}

function start() {
  kill_all

  # Install dependencies and build the Frontend
  cd "$scripts_dir/../../frontend/webapp"
  echo "Odigos UI - ⏳ Frontend installing"
  yarn install > /dev/null 2> "$log_filename"
  echo "Odigos UI - ⏳ Frontend building"
  yarn build > /dev/null 2> "$log_filename"

  # Build and start the Backend
  cd "../"
  echo "Odigos UI - ⏳ Backend building"
  go build -o ./odigos-backend > /dev/null 2> "$log_filename"
  echo "Odigos UI - ⏳ Backend starting"
  ./odigos-backend --port 8085 --debug --address 0.0.0.0 > /dev/null 2> "$log_filename" &
  sleep 3
  echo $! > "$back_pid_filename"
  back_pid=$(get_process_id "$back_pid_filename")
  check_process $back_pid "Backend" "$log_filename"

  # Start the Frontend
  # (we could skip this step, and simply use the UI on port 3001 from the Backend build - but we may want to run tests on the UI in real-time while developing, hence we will use port 3000 from the Frontend build)
  cd "./webapp"
  echo "Odigos UI - ⏳ Frontend starting"
  yarn dev > /dev/null 2> "$log_filename" &
  sleep 3
  echo $! > "$front_pid_filename"
  front_pid=$(get_process_id "$front_pid_filename")
  check_process $front_pid "Frontend" "$log_filename"
}

function test() {
  # Run tests on the Frontend
  cd "$scripts_dir/../../frontend/webapp"
  echo "Odigos UI - 👀 Frontend testing"

  set +e # Temporarily disable "exit on error"
  yarn cy:run
  test_exit_code=$?
  set -e # Re-enable "exit on error"

  if [ $test_exit_code -ne 0 ]; then
    echo "Odigos UI - ❌ Frontend tests failed"
  else
    echo "Odigos UI - ✅ Frontend tests passed"
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
