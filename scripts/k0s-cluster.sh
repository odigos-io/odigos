#!/usr/bin/env bash
# Manage a local 2-node k0s cluster in Docker (kind replacement).
# Full upstream Kubernetes via k0s — works on macOS (Docker Desktop/OrbStack/Colima) and Linux/CI.
#
# Usage:
#   scripts/k0s-cluster.sh create
#   scripts/k0s-cluster.sh delete
#   scripts/k0s-cluster.sh load IMAGE_OR_ARCHIVE [IMAGE_OR_ARCHIVE...]
#   scripts/k0s-cluster.sh kubeconfig
#
# Env:
#   K0S_IMAGE       - k0s OCI image (default: docker.io/k0sproject/k0s:v1.32.13-k0s.0)
#   K0S_CONTROLLER  - controller container/hostname (default: k0s-controller)
#   K0S_WORKER      - worker container/hostname (default: k0s-worker1)
#   K0S_API_PORT    - host port for API server (default: 6443)
#   KUBECONFIG      - kubeconfig path to write (default: ~/.kube/config)

set -euo pipefail

K0S_IMAGE="${K0S_IMAGE:-docker.io/k0sproject/k0s:v1.32.13-k0s.0}"
K0S_CONTROLLER="${K0S_CONTROLLER:-k0s-controller}"
K0S_WORKER="${K0S_WORKER:-k0s-worker1}"
K0S_API_PORT="${K0S_API_PORT:-6443}"
KUBECONFIG="${KUBECONFIG:-${HOME}/.kube/config}"

log() { echo "[k0s-cluster] $*"; }

die() { echo "[k0s-cluster] ERROR: $*" >&2; exit 1; }

container_exists() {
  docker inspect "$1" >/dev/null 2>&1
}

delete_cluster() {
  log "Deleting k0s cluster containers (if any)"
  docker rm -f "${K0S_WORKER}" "${K0S_CONTROLLER}" >/dev/null 2>&1 || true
}

write_kubeconfig() {
  mkdir -p "$(dirname "${KUBECONFIG}")"
  local tmp
  tmp="$(mktemp)"
  docker exec "${K0S_CONTROLLER}" k0s kubeconfig admin >"${tmp}"
  # Point the API server at the published host port.
  if [[ "$(uname -s)" == "Darwin" ]]; then
    sed -i '' -E "s|https://[^:]+:6443|https://127.0.0.1:${K0S_API_PORT}|" "${tmp}"
  else
    sed -i -E "s|https://[^:]+:6443|https://127.0.0.1:${K0S_API_PORT}|" "${tmp}"
  fi
  mv "${tmp}" "${KUBECONFIG}"
  log "Wrote kubeconfig to ${KUBECONFIG}"
}

wait_for_api() {
  local i
  for i in $(seq 1 60); do
    if docker exec "${K0S_CONTROLLER}" k0s kubectl get nodes >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  die "timed out waiting for k0s API on ${K0S_CONTROLLER}"
}

wait_for_nodes() {
  local expected="$1"
  local i ready
  for i in $(seq 1 90); do
    ready="$(docker exec "${K0S_CONTROLLER}" k0s kubectl get nodes --no-headers 2>/dev/null | awk '$2=="Ready"{c++} END{print c+0}')"
    if [[ "${ready}" -ge "${expected}" ]]; then
      docker exec "${K0S_CONTROLLER}" k0s kubectl get nodes -o wide
      return 0
    fi
    sleep 2
  done
  docker exec "${K0S_CONTROLLER}" k0s kubectl get nodes -o wide || true
  die "timed out waiting for ${expected} Ready nodes (have ${ready:-0})"
}

# Args after name are docker run args: [OPTIONS...] IMAGE [COMMAND...]
# Callers must place ${K0S_IMAGE} before the container command.
run_node() {
  local name="$1"
  shift
  docker run -d \
    --name "${name}" \
    --hostname "${name}" \
    --privileged \
    -v /var/lib/k0s \
    -v /var/log/pods \
    --tmpfs /run \
    --tmpfs /tmp \
    "$@"
}

create_cluster() {
  delete_cluster

  log "Creating controller+worker node ${K0S_CONTROLLER} (image=${K0S_IMAGE})"
  run_node "${K0S_CONTROLLER}" \
    -p "${K0S_API_PORT}:6443" \
    "${K0S_IMAGE}" \
    k0s controller --enable-worker >/dev/null

  wait_for_api
  wait_for_nodes 1

  log "Joining additional worker ${K0S_WORKER}"
  local token
  token="$(docker exec "${K0S_CONTROLLER}" k0s token create --role=worker)"
  run_node "${K0S_WORKER}" \
    "${K0S_IMAGE}" \
    k0s worker "${token}" >/dev/null

  wait_for_nodes 2
  write_kubeconfig
  log "k0s cluster ready (nodes: ${K0S_CONTROLLER}, ${K0S_WORKER})"
}

import_into_node() {
  local node="$1"
  local src="$2"
  if [[ -f "${src}" ]]; then
    log "Importing archive ${src} into ${node}"
    docker exec -i "${node}" k0s ctr images import - <"${src}"
  else
    log "Importing image ${src} into ${node}"
    docker save "${src}" | docker exec -i "${node}" k0s ctr images import -
  fi
}

load_images() {
  [[ "$#" -ge 1 ]] || die "load requires at least one image or archive"
  container_exists "${K0S_CONTROLLER}" || die "cluster not running (missing ${K0S_CONTROLLER})"
  container_exists "${K0S_WORKER}" || die "cluster not running (missing ${K0S_WORKER})"

  local src
  for src in "$@"; do
    import_into_node "${K0S_CONTROLLER}" "${src}"
    import_into_node "${K0S_WORKER}" "${src}"
  done
}

usage() {
  cat <<EOF
Usage: $0 {create|delete|load|kubeconfig} [args...]

  create              Start a 2-node k0s-in-Docker cluster and write kubeconfig
  delete              Remove cluster containers
  load IMG|TAR ...    Import local docker images/archives into both nodes
  kubeconfig          Refresh kubeconfig from the running controller

Environment:
  K0S_IMAGE=${K0S_IMAGE}
  K0S_CONTROLLER=${K0S_CONTROLLER}
  K0S_WORKER=${K0S_WORKER}
  K0S_API_PORT=${K0S_API_PORT}
  KUBECONFIG=${KUBECONFIG}
EOF
}

main() {
  [[ "$#" -ge 1 ]] || { usage; exit 1; }
  local cmd="$1"
  shift
  case "${cmd}" in
    create) create_cluster ;;
    delete) delete_cluster ;;
    load) load_images "$@" ;;
    kubeconfig) write_kubeconfig ;;
    -h|--help|help) usage ;;
    *) usage; die "unknown command: ${cmd}" ;;
  esac
}

main "$@"
