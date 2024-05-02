# #!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
SCRIPT_ROOT="${SCRIPT_DIR}/.."
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

source "${CODEGEN_PKG}/kube_codegen.sh"

THIS_PKG="k8s.io/code-generator/examples"

kube::codegen::gen_client \
    --with-watch \
    --with-applyconfig \
    --one-input-api "actions/v1alpha1" \
    --output-dir "${SCRIPT_ROOT}/generated/actions" \
    --output-pkg "github.com/keyval-dev/odigos/api/generated/actions" \
    --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
    "${SCRIPT_ROOT}"

kube::codegen::gen_client \
    --with-watch \
    --with-applyconfig \
    --one-input-api "odigos/v1alpha1" \
    --output-dir "${SCRIPT_ROOT}/generated/odigos" \
    --output-pkg "github.com/keyval-dev/odigos/api/generated/odigos" \
    --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.go.txt" \
    "${SCRIPT_ROOT}"