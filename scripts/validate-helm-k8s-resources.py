#!/usr/bin/env python3
"""
Linter to validate Kubernetes workloads in Helm charts have proper configuration:
- Resource limits and requests (cpu and memory)
- GOMAXPROCS and GOMEMLIMIT for Go containers
- ImagePullSecrets support (when --require-image-pull-secrets is set)

Usage:
    python validate-helm-k8s-resources.py <rendered_manifests.yaml>

    # Or pipe helm template output:
    helm template myrelease ./helm/odigos | python validate-helm-k8s-resources.py -

    # Check that imagePullSecrets are rendered (for templates that should support it):
    helm template myrelease ./helm/odigos --set imagePullSecrets[0]=my-secret | \
        python validate-helm-k8s-resources.py - --require-image-pull-secrets
"""

import argparse
import re
import sys
import yaml
from dataclasses import dataclass
from typing import Any


@dataclass
class ContainerChecks:
    """Configuration for what to check on a container."""
    check_resources: bool = True          # Check for resource limits/requests
    check_gomaxprocs: bool = False        # Check for GOMAXPROCS env var
    check_gomemlimit: bool = False        # Check for GOMEMLIMIT env var
    is_go_container: bool = False         # Container runs Go code


# Container configuration registry
# Format: (workload_name_pattern, container_name_pattern) -> ContainerChecks
CONTAINER_CONFIG: list[tuple[str, str, ContainerChecks]] = [
    # ============ odigos chart - Go containers ============
    (r"^odigos-autoscaler$", r"^manager$", ContainerChecks(
        check_resources=True, check_gomaxprocs=True, check_gomemlimit=True, is_go_container=True
    )),
    (r"^odigos-scheduler$", r"^manager$", ContainerChecks(
        check_resources=True, check_gomaxprocs=True, check_gomemlimit=True, is_go_container=True
    )),
    (r"^odigos-instrumentor$", r"^manager$", ContainerChecks(
        check_resources=True, check_gomaxprocs=True, check_gomemlimit=True, is_go_container=True
    )),
    (r"^odigos-ui$", r"^ui$", ContainerChecks(
        check_resources=True, check_gomaxprocs=True, check_gomemlimit=True, is_go_container=True
    )),
    (r"^central-proxy$", r"^central-proxy$", ContainerChecks(
        check_resources=True, check_gomaxprocs=True, check_gomemlimit=True, is_go_container=True
    )),
    (r"^odiglet$", r"^odiglet$", ContainerChecks(
        check_resources=True, check_gomaxprocs=True, check_gomemlimit=True, is_go_container=True
    )),
    (r"^odiglet$", r"^data-collection$", ContainerChecks(
        check_resources=True, check_gomaxprocs=True, check_gomemlimit=True, is_go_container=True
    )),
    (r"^odiglet$", r"^deviceplugin$", ContainerChecks(
        check_resources=True, check_gomaxprocs=True, check_gomemlimit=True, is_go_container=True
    )),
    (r"^odiglet$", r"^init$", ContainerChecks(
        check_resources=True, check_gomaxprocs=True, check_gomemlimit=True, is_go_container=True
    )),
    (r"^cleanup-job$", r"^cleanup$", ContainerChecks(
        check_resources=True, check_gomaxprocs=True, check_gomemlimit=True, is_go_container=True
    )),

    # ============ odigos-central chart - Go containers ============
    (r"^central-backend$", r"^central-backend$", ContainerChecks(
        check_resources=True, check_gomaxprocs=True, check_gomemlimit=True, is_go_container=True
    )),
    (r"^central-ui$", r"^central-ui$", ContainerChecks(
        check_resources=True, check_gomaxprocs=True, check_gomemlimit=True, is_go_container=True
    )),

    # ============ Third-party / non-Go containers ============
    # VictoriaMetrics - third-party Go app, manages its own memory
    (r"^odigos-victoriametrics$", r"^vm$", ContainerChecks(
        check_resources=True, check_gomaxprocs=False, check_gomemlimit=False, is_go_container=False
    )),
    # Keycloak - Java application
    (r"^keycloak$", r".*", ContainerChecks(
        check_resources=True, check_gomaxprocs=False, check_gomemlimit=False, is_go_container=False
    )),
    # Redis - C application
    (r"^redis$", r".*", ContainerChecks(
        check_resources=True, check_gomaxprocs=False, check_gomemlimit=False, is_go_container=False
    )),
    # Image pull init container - just pulls image, doesn't run code
    (r"^odiglet$", r"^odigos-agents-image-pull$", ContainerChecks(
        check_resources=True, check_gomaxprocs=False, check_gomemlimit=False, is_go_container=False
    )),
]

# Workloads that use third-party images and don't need imagePullSecrets from Odigos registry
THIRD_PARTY_IMAGE_WORKLOADS = [
    r"^odigos-victoriametrics$",  # Uses public VictoriaMetrics image
]

# Workload kinds to check
WORKLOAD_KINDS = {"Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob"}


@dataclass
class LintError:
    workload_kind: str
    workload_name: str
    container_name: str
    category: str
    message: str

    def __str__(self):
        if self.container_name:
            return f"[{self.category}] {self.workload_kind}/{self.workload_name} container={self.container_name}: {self.message}"
        return f"[{self.category}] {self.workload_kind}/{self.workload_name}: {self.message}"


def parse_memory_value(value: str) -> int | None:
    """Parse a Kubernetes memory value to bytes."""
    if not value:
        return None

    value = str(value).strip()

    # Handle pure numbers (bytes)
    if value.isdigit():
        return int(value)

    # Parse with units
    match = re.match(r'^(\d+(?:\.\d+)?)\s*([EPTGMK]i?|[eptgmk])?[bB]?$', value)
    if not match:
        return None

    num = float(match.group(1))
    unit = match.group(2) or ''

    multipliers = {
        '': 1,
        'K': 1000,
        'Ki': 1024,
        'M': 1000**2,
        'Mi': 1024**2,
        'G': 1000**3,
        'Gi': 1024**3,
        'T': 1000**4,
        'Ti': 1024**4,
        'P': 1000**5,
        'Pi': 1024**5,
        'E': 1000**6,
        'Ei': 1024**6,
    }

    return int(num * multipliers.get(unit, 1))


def parse_gomemlimit_value(value: str) -> int | None:
    """Parse a GOMEMLIMIT value to bytes."""
    if not value:
        return None

    value = str(value).strip()

    # Handle pure numbers (bytes)
    if value.isdigit():
        return int(value)

    # GOMEMLIMIT uses B suffix (e.g., MiB, GiB)
    match = re.match(r'^(\d+(?:\.\d+)?)\s*([EPTGMK]i)?[bB]$', value)
    if not match:
        # Try without B suffix
        match = re.match(r'^(\d+(?:\.\d+)?)\s*([EPTGMK]i)?$', value)
        if not match:
            return None

    num = float(match.group(1))
    unit = match.group(2) or ''

    multipliers = {
        '': 1,
        'Ki': 1024,
        'Mi': 1024**2,
        'Gi': 1024**3,
        'Ti': 1024**4,
        'Pi': 1024**5,
        'Ei': 1024**6,
    }

    return int(num * multipliers.get(unit, 1))


def get_container_config(workload_name: str, container_name: str) -> ContainerChecks | None:
    """Get the check configuration for a container."""
    for pattern, container_pattern, config in CONTAINER_CONFIG:
        if re.match(pattern, workload_name) and re.match(container_pattern, container_name):
            return config
    return None


def get_env_var(env_list: list, name: str) -> dict | None:
    """Get an environment variable from a list of env vars."""
    if not env_list:
        return None
    for env in env_list:
        if env.get("name") == name:
            return env
    return None


def validate_resources(
    container: dict,
    workload_kind: str,
    workload_name: str,
    container_name: str
) -> list[LintError]:
    """Validate that a container has proper resource limits and requests."""
    errors = []
    resources = container.get("resources", {})

    if not resources:
        errors.append(LintError(
            workload_kind, workload_name, container_name,
            "resources", "No resources defined"
        ))
        return errors

    limits = resources.get("limits", {})
    requests = resources.get("requests", {})

    # Check limits
    if not limits:
        errors.append(LintError(
            workload_kind, workload_name, container_name,
            "resources", "No resource limits defined"
        ))
    else:
        if "cpu" not in limits:
            errors.append(LintError(
                workload_kind, workload_name, container_name,
                "resources", "No CPU limit defined"
            ))
        if "memory" not in limits:
            errors.append(LintError(
                workload_kind, workload_name, container_name,
                "resources", "No memory limit defined"
            ))

    # Check requests
    if not requests:
        errors.append(LintError(
            workload_kind, workload_name, container_name,
            "resources", "No resource requests defined"
        ))
    else:
        if "cpu" not in requests:
            errors.append(LintError(
                workload_kind, workload_name, container_name,
                "resources", "No CPU request defined"
            ))
        if "memory" not in requests:
            errors.append(LintError(
                workload_kind, workload_name, container_name,
                "resources", "No memory request defined"
            ))

    return errors


def validate_gomaxprocs(
    env_var: dict | None,
    container_name: str,
    has_cpu_limit: bool,
    workload_kind: str,
    workload_name: str
) -> LintError | None:
    """Validate GOMAXPROCS configuration."""
    if not has_cpu_limit:
        if not env_var:
            return None
        # If set, it should still be via resourceFieldRef

    if not env_var:
        if has_cpu_limit:
            return LintError(
                workload_kind, workload_name, container_name,
                "GOMAXPROCS", "GOMAXPROCS not set but container has CPU limit"
            )
        return None

    value_from = env_var.get("valueFrom")
    if not value_from:
        value = env_var.get("value")
        if value:
            return LintError(
                workload_kind, workload_name, container_name,
                "GOMAXPROCS", f"GOMAXPROCS set to static value '{value}' instead of resourceFieldRef"
            )
        return LintError(
            workload_kind, workload_name, container_name,
            "GOMAXPROCS", "GOMAXPROCS has no value or valueFrom"
        )

    resource_field_ref = value_from.get("resourceFieldRef")
    if not resource_field_ref:
        return LintError(
            workload_kind, workload_name, container_name,
            "GOMAXPROCS", f"GOMAXPROCS should use resourceFieldRef, not {list(value_from.keys())}"
        )

    resource = resource_field_ref.get("resource", "")
    if resource != "limits.cpu":
        return LintError(
            workload_kind, workload_name, container_name,
            "GOMAXPROCS", f"GOMAXPROCS resourceFieldRef should reference 'limits.cpu', not '{resource}'"
        )

    return None


def validate_gomemlimit(
    env_var: dict | None,
    memory_limit_bytes: int | None,
    workload_kind: str,
    workload_name: str,
    container_name: str
) -> LintError | None:
    """Validate GOMEMLIMIT configuration."""
    if not memory_limit_bytes:
        if not env_var:
            return None
        return None

    if not env_var:
        return LintError(
            workload_kind, workload_name, container_name,
            "GOMEMLIMIT", "GOMEMLIMIT not set but container has memory limit"
        )

    value = env_var.get("value")
    value_from = env_var.get("valueFrom")

    if value_from:
        return None

    if not value:
        return LintError(
            workload_kind, workload_name, container_name,
            "GOMEMLIMIT", "GOMEMLIMIT has no value"
        )

    # Handle Helm template expressions
    if "{{" in str(value) or "}}" in str(value):
        return None

    gomemlimit_bytes = parse_gomemlimit_value(value)
    if gomemlimit_bytes is None:
        return LintError(
            workload_kind, workload_name, container_name,
            "GOMEMLIMIT", f"GOMEMLIMIT has unparseable value: '{value}'"
        )

    expected_min = int(memory_limit_bytes * 0.70)
    expected_max = int(memory_limit_bytes * 0.90)

    if gomemlimit_bytes < expected_min or gomemlimit_bytes > expected_max:
        expected = int(memory_limit_bytes * 0.80)
        return LintError(
            workload_kind, workload_name, container_name,
            "GOMEMLIMIT", f"GOMEMLIMIT ({value}) is not within 70-90% of memory limit. Expected ~{expected} bytes, got {gomemlimit_bytes} bytes"
        )

    return None


def is_third_party_image_workload(workload_name: str) -> bool:
    """Check if a workload uses third-party images."""
    for pattern in THIRD_PARTY_IMAGE_WORKLOADS:
        if re.match(pattern, workload_name):
            return True
    return False


def validate_image_pull_secrets(
    spec: dict,
    workload_kind: str,
    workload_name: str
) -> LintError | None:
    """Validate that imagePullSecrets is configured for Odigos images."""
    # Skip workloads that use third-party images
    if is_third_party_image_workload(workload_name):
        return None

    image_pull_secrets = spec.get("imagePullSecrets")

    if not image_pull_secrets:
        return LintError(
            workload_kind, workload_name, "",
            "imagePullSecrets", "imagePullSecrets not rendered (template may not support imagePullSecrets)"
        )

    return None


def get_containers(spec: dict) -> list[tuple[str, dict, bool]]:
    """Get all containers from a pod spec."""
    containers = []

    for container in spec.get("initContainers", []):
        containers.append((container.get("name", "unknown"), container, True))

    for container in spec.get("containers", []):
        containers.append((container.get("name", "unknown"), container, False))

    return containers


def validate_workload(doc: dict, require_image_pull_secrets: bool = False) -> list[LintError]:
    """Validate a single Kubernetes workload document."""
    errors = []

    kind = doc.get("kind", "")
    if kind not in WORKLOAD_KINDS:
        return errors

    metadata = doc.get("metadata", {})
    name = metadata.get("name", "unknown")

    # Get the pod template spec
    spec = doc.get("spec", {})

    if kind == "CronJob":
        spec = spec.get("jobTemplate", {}).get("spec", {}).get("template", {}).get("spec", {})
    elif kind == "Job":
        spec = spec.get("template", {}).get("spec", {})
    else:  # Deployment, StatefulSet, DaemonSet
        spec = spec.get("template", {}).get("spec", {})

    if not spec:
        return errors

    # Check imagePullSecrets at pod level (only when required)
    if require_image_pull_secrets:
        error = validate_image_pull_secrets(spec, kind, name)
        if error:
            errors.append(error)

    containers = get_containers(spec)

    for container_name, container, is_init in containers:
        config = get_container_config(name, container_name)
        if config is None:
            # Unknown container - skip
            continue

        resources = container.get("resources", {})
        limits = resources.get("limits", {})
        requests = resources.get("requests", {})

        # Check resources
        if config.check_resources:
            resource_errors = validate_resources(container, kind, name, container_name)
            errors.extend(resource_errors)

        # Get limits for GOMAXPROCS/GOMEMLIMIT checks
        memory_limit = limits.get("memory") or requests.get("memory")
        cpu_limit = limits.get("cpu") or requests.get("cpu")
        memory_limit_bytes = parse_memory_value(memory_limit) if memory_limit else None
        has_cpu_limit = cpu_limit is not None

        env_list = container.get("env", [])

        # Check GOMEMLIMIT
        if config.check_gomemlimit:
            gomemlimit_env = get_env_var(env_list, "GOMEMLIMIT")
            error = validate_gomemlimit(gomemlimit_env, memory_limit_bytes, kind, name, container_name)
            if error:
                errors.append(error)

        # Check GOMAXPROCS
        if config.check_gomaxprocs:
            gomaxprocs_env = get_env_var(env_list, "GOMAXPROCS")
            error = validate_gomaxprocs(gomaxprocs_env, container_name, has_cpu_limit, kind, name)
            if error:
                errors.append(error)

    return errors


def load_yaml_documents(content: str) -> list[dict]:
    """Load all YAML documents from a string."""
    docs = []
    for doc in yaml.safe_load_all(content):
        if doc:
            docs.append(doc)
    return docs


def main():
    parser = argparse.ArgumentParser(
        description="Validate Kubernetes workloads in Helm charts"
    )
    parser.add_argument(
        "file",
        nargs="?",
        default="-",
        help="YAML file to validate (use '-' for stdin)",
    )
    parser.add_argument(
        "--strict",
        action="store_true",
        help="Exit with error if any issues are found",
    )
    parser.add_argument(
        "--require-image-pull-secrets",
        action="store_true",
        help="Require imagePullSecrets to be present (use when testing with imagePullSecrets values set)",
    )
    parser.add_argument(
        "--category",
        action="append",
        dest="categories",
        help="Only check specific categories (resources, GOMAXPROCS, GOMEMLIMIT, imagePullSecrets)",
    )

    args = parser.parse_args()

    # Read input
    if args.file == "-":
        content = sys.stdin.read()
    else:
        with open(args.file, "r") as f:
            content = f.read()

    # Parse YAML
    try:
        docs = load_yaml_documents(content)
    except yaml.YAMLError as e:
        print(f"Error parsing YAML: {e}", file=sys.stderr)
        sys.exit(1)

    # Validate each document
    all_errors: list[LintError] = []
    for doc in docs:
        errors = validate_workload(doc, require_image_pull_secrets=args.require_image_pull_secrets)
        all_errors.extend(errors)

    # Filter by category if specified
    if args.categories:
        all_errors = [e for e in all_errors if e.category in args.categories]

    # Report results
    if all_errors:
        # Group errors by category for better readability
        by_category: dict[str, list[LintError]] = {}
        for error in all_errors:
            by_category.setdefault(error.category, []).append(error)

        print(f"Found {len(all_errors)} issue(s):\n", file=sys.stderr)
        for category, errors in sorted(by_category.items()):
            print(f"  {category} ({len(errors)} issues):", file=sys.stderr)
            for error in errors:
                print(f"    - {error.workload_kind}/{error.workload_name}", end="", file=sys.stderr)
                if error.container_name:
                    print(f" container={error.container_name}", end="", file=sys.stderr)
                print(f": {error.message}", file=sys.stderr)
            print(file=sys.stderr)

        if args.strict:
            sys.exit(1)
    else:
        print("All workloads pass validation.")

    return 0


if __name__ == "__main__":
    sys.exit(main())
