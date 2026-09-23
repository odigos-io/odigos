#!/usr/bin/env bash
# Resolve the container image repository (registry/name, no tag) for local
# make deploy / load-to-kind workflows.
#
# Enterprise installs (odigos-pro secret or ODIGOS_TIER=onprem) default to
# registry.odigos.io and odigos-enterprise-* names where those differ from OSS.
# When a matching workload already exists in the cluster with the expected image
# name, its registry/prefix is preserved (custom imagePrefix, mirrors, etc.).
#
# Usage: resolve-dev-image.sh <component>
# Env:
#   ODIGOS_NS          - namespace (default: odigos-system)
#   ODIGOS_ENTERPRISE  - true|false to skip auto-detection
#   ORG_OVERRIDE       - force registry/org (when user set ORG on the CLI/env)
#   OSS_ORG            - default OSS registry (default: docker.io/keyval)
#   ENTERPRISE_ORG     - default enterprise registry (default: registry.odigos.io)
#   IMG_SUFFIX         - optional image suffix (e.g. -rhel-certified)

set -euo pipefail

component="${1:?component name required (e.g. autoscaler, ui, odiglet)}"
ns="${ODIGOS_NS:-odigos-system}"
img_suffix="${IMG_SUFFIX:-}"
oss_org="${OSS_ORG:-docker.io/keyval}"
enterprise_org="${ENTERPRISE_ORG:-registry.odigos.io}"

is_enterprise() {
	case "${ODIGOS_ENTERPRISE:-}" in
		true|1|yes) return 0 ;;
		false|0|no) return 1 ;;
	esac
	local kopts=(--request-timeout=3s)
	if kubectl "${kopts[@]}" get secret odigos-pro -n "${ns}" >/dev/null 2>&1; then
		return 0
	fi
	tier="$(kubectl "${kopts[@]}" get cm odigos-deployment -n "${ns}" -o jsonpath='{.data.ODIGOS_TIER}' 2>/dev/null || true)"
	[ "${tier}" = "onprem" ]
}

# Image repository names that gain an "enterprise-" infix on enterprise installs.
expected_image_name() {
	local c="$1"
	if is_enterprise; then
		case "${c}" in
			ui|instrumentor|odiglet|collector|agents)
				echo "odigos-enterprise-${c}${img_suffix}"
				return
				;;
		esac
	fi
	echo "odigos-${c}${img_suffix}"
}

fallback_org() {
	if [ -n "${ORG_OVERRIDE:-}" ]; then
		echo "${ORG_OVERRIDE}"
		return
	fi
	if is_enterprise; then
		echo "${enterprise_org}"
	else
		echo "${oss_org}"
	fi
}

# Strip digest and tag, leaving registry/repo.
repo_without_tag() {
	local img="${1%%@*}"
	local name="${img##*/}"
	case "${name}" in
		*:*) echo "${img%:*}" ;;
		*) echo "${img}" ;;
	esac
}

image_basename() {
	local repo="$1"
	echo "${repo##*/}"
}

lookup_cluster_image() {
	local kopts=(--request-timeout=3s)
	case "$1" in
		autoscaler)
			kubectl "${kopts[@]}" get deploy odigos-autoscaler -n "${ns}" \
				-o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || true
			;;
		scheduler)
			kubectl "${kopts[@]}" get deploy odigos-scheduler -n "${ns}" \
				-o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || true
			;;
		instrumentor)
			kubectl "${kopts[@]}" get deploy odigos-instrumentor -n "${ns}" \
				-o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || true
			;;
		ui)
			kubectl "${kopts[@]}" get deploy odigos-ui -n "${ns}" \
				-o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || true
			;;
		odiglet)
			kubectl "${kopts[@]}" get ds odiglet -n "${ns}" \
				-o jsonpath='{.spec.template.spec.containers[?(@.name=="odiglet")].image}' 2>/dev/null || true
			;;
		collector)
			local img
			img="$(kubectl "${kopts[@]}" get deploy odigos-gateway -n "${ns}" \
				-o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || true)"
			if [ -z "${img}" ]; then
				img="$(kubectl "${kopts[@]}" get ds odiglet -n "${ns}" \
					-o jsonpath='{.spec.template.spec.containers[?(@.name=="data-collection")].image}' 2>/dev/null || true)"
			fi
			printf '%s' "${img}"
			;;
		agents)
			kubectl "${kopts[@]}" get ds odiglet -n "${ns}" \
				-o jsonpath='{.spec.template.spec.initContainers[?(@.name=="odigos-agents-image-pull")].image}' 2>/dev/null || true
			;;
		*)
			printf ''
			;;
	esac
}

expected="$(expected_image_name "${component}")"
fallback="$(fallback_org)/${expected}"

cluster_img="$(lookup_cluster_image "${component}")"
if [ -n "${cluster_img}" ]; then
	cluster_repo="$(repo_without_tag "${cluster_img}")"
	cluster_base="$(image_basename "${cluster_repo}")"
	# Only reuse the live image when its name matches what this tier expects.
	# That preserves custom imagePrefix/mirrors while recovering from a prior
	# deploy that tagged the wrong OSS name into an enterprise cluster.
	if [ "${cluster_base}" = "${expected}" ]; then
		# Shared components (autoscaler/scheduler) keep the same image name on
		# enterprise, so a prior OSS `make deploy` may have left docker.io/keyval
		# in the cluster — treat that as stale and use the enterprise registry.
		if is_enterprise; then
			case "${cluster_repo}" in
				"${oss_org}"/*)
					echo "$(fallback_org)/${expected}"
					exit 0
					;;
			esac
		fi
		echo "${cluster_repo}"
		exit 0
	fi
fi

echo "${fallback}"
