#!/usr/bin/env bash
# Generate a merged /licenses tree (superset) for RHEL-certified images.
# Usage:
#   ./scripts/generate-rhel-licenses.sh <output-dir> [enterprise-repo-root]
#
# OSS modules are taken from this repo. When enterprise-repo-root is set, enterprise
# module licenses are merged in as well (typically odigos-enterprise @ main).
set -euo pipefail

OUT_DIR="${1:?output directory required}"
ENTERPRISE_ROOT="${2:-}"
OSS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

merge_tree() {
  local src="$1"
  if [[ -d "${src}" ]]; then
    # go-licenses trees use module paths as dirs; last write wins for duplicates.
    cp -a "${src}/." "${OUT_DIR}/"
  fi
}

merge_license_file() {
  local file="$1"
  if [[ -f "${file}" ]]; then
    cp "${file}" "${OUT_DIR}/LICENSE"
  fi
}

run_make_licenses() {
  local dir="$1"
  echo "==> make licenses in ${dir}"
  (cd "${dir}" && make licenses)
}

rm -rf "${OUT_DIR}"
mkdir -p "${OUT_DIR}"

# Project-level Apache license (also overwritten by per-module LICENSE copies).
merge_license_file "${OSS_ROOT}/LICENSE"

OSS_MAKE_LICENSE_MODULES=(
  autoscaler
  scheduler
  instrumentor
  collector
  odiglet
  frontend
  operator
  cli
)

for mod in "${OSS_MAKE_LICENSE_MODULES[@]}"; do
  run_make_licenses "${OSS_ROOT}/${mod}"
done

# Collector writes into odigosotelcol/licenses; others write <module>/licenses.
merge_tree "${OSS_ROOT}/collector/odigosotelcol/licenses"
merge_license_file "${OSS_ROOT}/collector/LICENSE"

for mod in autoscaler scheduler instrumentor odiglet frontend operator cli; do
  merge_tree "${OSS_ROOT}/${mod}/licenses"
  merge_license_file "${OSS_ROOT}/${mod}/LICENSE"
done

if [[ -n "${ENTERPRISE_ROOT}" ]]; then
  if [[ ! -d "${ENTERPRISE_ROOT}" ]]; then
    echo "enterprise root not found: ${ENTERPRISE_ROOT}" >&2
    exit 1
  fi

  # Ensure private Go modules resolve when generating enterprise licenses.
  if [[ -n "${GITHUB_TOKEN:-}" ]]; then
    git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"
    export GOPRIVATE="${GOPRIVATE:-github.com/odigos-io/*}"
  fi

  ENTERPRISE_MAKE_LICENSE_MODULES=(
    central-backend
    central-proxy
    central-ui
    collector
  )

  for mod in "${ENTERPRISE_MAKE_LICENSE_MODULES[@]}"; do
    if [[ -f "${ENTERPRISE_ROOT}/${mod}/Makefile" ]]; then
      run_make_licenses "${ENTERPRISE_ROOT}/${mod}" || {
        echo "warning: make licenses failed for enterprise/${mod}; continuing" >&2
      }
    fi
  done

  # Enterprise collector merges odigosotelcol (+ opamp supervisor) licenses.
  merge_tree "${ENTERPRISE_ROOT}/collector/odigosotelcol/licenses"
  merge_license_file "${ENTERPRISE_ROOT}/collector/LICENSE"

  for mod in central-backend central-proxy central-ui; do
    merge_tree "${ENTERPRISE_ROOT}/${mod}/licenses"
    merge_license_file "${ENTERPRISE_ROOT}/${mod}/LICENSE"
  done

  # Enterprise odiglet vendors a checked-in licenses/ tree used by its image build.
  merge_tree "${ENTERPRISE_ROOT}/odiglet/licenses"

  for mod in instrumentor ui insights odiglet; do
    merge_license_file "${ENTERPRISE_ROOT}/${mod}/LICENSE"
  done

  # Best-effort go-licenses for enterprise Go modules that lack a make target.
  if command -v go >/dev/null 2>&1; then
    TOOLS_BIN="${OSS_ROOT}/bin"
    mkdir -p "${TOOLS_BIN}"
    GOBIN="${TOOLS_BIN}" go install github.com/google/go-licenses@latest
    for mod in instrumentor ui insights odiglet; do
      if [[ -f "${ENTERPRISE_ROOT}/${mod}/go.mod" ]]; then
        echo "==> go-licenses save enterprise/${mod}"
        tmp="$(mktemp -d)"
        if (
          cd "${ENTERPRISE_ROOT}/${mod}" &&
          "${TOOLS_BIN}/go-licenses" save . --save_path="${tmp}" \
            --ignore "github.com/odigos-io/odigos-enterprise" \
            --ignore "github.com/odigos-io/${mod}"
        ); then
          merge_tree "${tmp}"
        else
          echo "warning: go-licenses failed for enterprise/${mod}; continuing" >&2
        fi
        rm -rf "${tmp}"
      fi
    done
  fi
fi

# Ensure /licenses/LICENSE exists for preflight.
if [[ ! -f "${OUT_DIR}/LICENSE" ]]; then
  merge_license_file "${OSS_ROOT}/LICENSE"
fi

echo "Wrote merged licenses to ${OUT_DIR}"
find "${OUT_DIR}" -type f | wc -l | awk '{print "license files:", $1}'
