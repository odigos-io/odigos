#!/bin/bash

run_cypress_test() {
  local spec=$1
  npx cypress run --spec "$spec"
  local status=$?

  if [ $status -ne 0 ]; then
    echo "Cypress tests failed"
    # Stop the background process
    cd ../../tests/e2e/fe-synthetic || exit
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

if [ "$2" = "action-addition" ]; then
  run_cypress_test "cypress/e2e/action-testing.cy.ts"
fi

echo "Cypress tests passed"
