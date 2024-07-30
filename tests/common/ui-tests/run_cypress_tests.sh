#!/bin/bash

echo "Running Cypress tests"
cd ../../../frontend/webapp || exit
npx cypress run

status_cypress=$?
if [ $status_cypress -ne 0 ]; then
  echo "Cypress tests failed"

  # Stop the background process
  kill "$(cat odigos-ui.pid)"
  rm odigos-ui.pid
  rm ../../odigos-ui.log

  exit $status_cypress
else
  echo "Cypress tests passed"
fi