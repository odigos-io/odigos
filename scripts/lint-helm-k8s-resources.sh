#!/bin/bash
#
# Lint Helm charts to ensure Kubernetes workloads have proper configuration:
# - Resource limits and requests
# - GOMAXPROCS and GOMEMLIMIT for Go containers
# - imagePullSecrets support (when configured)
#
# Usage:
#   ./scripts/lint-helm-k8s-resources.sh
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
LINTER="${SCRIPT_DIR}/validate-helm-k8s-resources.py"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "=== Kubernetes Resource Configuration Linter ==="
echo ""

# Check if helm is installed
if ! command -v helm &> /dev/null; then
    echo -e "${RED}Error: helm is not installed${NC}" >&2
    exit 1
fi

# Check if python3 is installed
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}Error: python3 is not installed${NC}" >&2
    exit 1
fi

# Check if PyYAML is installed
if ! python3 -c "import yaml" &> /dev/null; then
    echo -e "${RED}Error: PyYAML is not installed. Install with: pip install pyyaml${NC}" >&2
    exit 1
fi

ERRORS=0
TEMP_DIR=$(mktemp -d)
trap "rm -rf ${TEMP_DIR}" EXIT

# Function to lint a helm chart
lint_chart() {
    local chart_path="$1"
    local chart_name="$(basename "${chart_path}")"
    local values_file="$2"
    local description="$3"
    local extra_args="${4:-}"

    echo -e "${YELLOW}Checking ${chart_name} (${description})...${NC}"

    local output_file="${TEMP_DIR}/${chart_name}-${description//[^a-zA-Z0-9]/-}.yaml"

    # Render the helm template
    if ! helm template test-release "${chart_path}" ${values_file:+--values "${values_file}"} > "${output_file}" 2>&1; then
        echo -e "${RED}  Failed to render helm template${NC}"
        cat "${output_file}" >&2
        ((ERRORS++))
        return 1
    fi

    # Run the linter
    if python3 "${LINTER}" --strict ${extra_args} "${output_file}"; then
        echo -e "${GREEN}  Passed${NC}"
    else
        ((ERRORS++))
    fi

    echo ""
}

# ==================== Test 1: Default values ====================
echo -e "${BLUE}=== Test 1: Basic validation (resources, GOMAXPROCS, GOMEMLIMIT) ===${NC}"
echo ""

lint_chart "${REPO_ROOT}/helm/odigos" "" "default values"

# ==================== Test 2: Enterprise mode ====================
echo -e "${BLUE}=== Test 2: Enterprise mode ===${NC}"
echo ""

ENTERPRISE_VALUES="${TEMP_DIR}/enterprise-values.yaml"
cat > "${ENTERPRISE_VALUES}" <<EOF
onPremToken: "test-token-for-linting"
clusterName: "test-cluster"
centralProxy:
  centralBackendURL: "https://example.com"
EOF
lint_chart "${REPO_ROOT}/helm/odigos" "${ENTERPRISE_VALUES}" "enterprise mode"

# ==================== Test 3: imagePullSecrets support ====================
echo -e "${BLUE}=== Test 3: imagePullSecrets support ===${NC}"
echo ""

PULL_SECRETS_VALUES="${TEMP_DIR}/pull-secrets-values.yaml"
cat > "${PULL_SECRETS_VALUES}" <<EOF
imagePullSecrets:
  - my-registry-secret
EOF
lint_chart "${REPO_ROOT}/helm/odigos" "${PULL_SECRETS_VALUES}" "with imagePullSecrets" "--require-image-pull-secrets"

# ==================== Test 4: odigos-central chart ====================
if [ -d "${REPO_ROOT}/helm/odigos-central" ]; then
    echo -e "${BLUE}=== Test 4: odigos-central chart ===${NC}"
    echo ""

    CENTRAL_VALUES="${TEMP_DIR}/central-values.yaml"
    cat > "${CENTRAL_VALUES}" <<EOF
onPremToken: "test-token-for-linting"
EOF
    lint_chart "${REPO_ROOT}/helm/odigos-central" "${CENTRAL_VALUES}" "default values"

    # Test imagePullSecrets in odigos-central
    CENTRAL_PULL_SECRETS="${TEMP_DIR}/central-pull-secrets.yaml"
    cat > "${CENTRAL_PULL_SECRETS}" <<EOF
onPremToken: "test-token-for-linting"
imagePullSecrets:
  - my-registry-secret
EOF
    lint_chart "${REPO_ROOT}/helm/odigos-central" "${CENTRAL_PULL_SECRETS}" "with imagePullSecrets" "--require-image-pull-secrets"
fi

# ==================== Summary ====================
echo "=== Summary ==="
if [ ${ERRORS} -eq 0 ]; then
    echo -e "${GREEN}All checks passed!${NC}"
    exit 0
else
    echo -e "${RED}${ERRORS} check(s) failed${NC}"
    exit 1
fi
