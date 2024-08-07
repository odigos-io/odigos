#!/bin/bash

run_cypress_test() {
  local spec=$1
  npx cypress run --spec "$spec"
  local status=$?

  if [ $status -ne 0 ]; then
    echo "Cypress tests failed"
    # Stop the background process
    kill "$(cat odigos-ui.pid)"
    rm odigos-ui.pid
    rm ../../odigos-ui.log
    exit $status
  fi
}

echo "Running Cypress tests"
cd ../../../frontend/webapp || exit

if [ "$1" = "include-onboarding-flow" ]; then
  run_cypress_test "cypress/e2e/onboarding-flow.cy.ts"
fi

run_cypress_test "cypress/e2e/test-overview.cy.ts"

echo "Cypress tests passed"
