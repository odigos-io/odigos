variable "ODIGOS_TAG" {
  default = "latest"
}

variable "IMAGE_SUFFIX" {
  default = ""
}

variable "GCP_REPO" {
  default = "us-central1-docker.pkg.dev/odigos-cloud/components"
}

variable "DEPOT_REPO" {
  default = "p0xd21zf5r.registry.depot.dev"
}

variable "DOCKERHUB_REPO" {
  default = "keyval"
}

variable "GHCR_REPO" {
  default = "ghcr.io/odigos-io"
}

group "modules" {
  targets = [
    "odigos-autoscaler",
    "odigos-scheduler",
    "odigos-instrumentor",
    "odigos-collector",
    "odigos-odiglet",
    "odigos-ui",
    "odigos-operator",
    "odigos-agents",
    "odigos-cli",
  ]
}

group "stack" {
  targets = [
    "odigos-ui",
    "odigos-collector",
    "odigos-odiglet",
    "odigos-autoscaler",
    "odigos-scheduler",
    "odigos-instrumentor",
    "odigos-agents",
    "odigos-cli",
  ]
}

group "modules-rhel" {
  targets = [
    "odigos-autoscaler-rhel",
    "odigos-scheduler-rhel",
    "odigos-instrumentor-rhel",
    "odigos-collector-rhel",
    "odigos-odiglet-rhel",
    "odigos-ui-rhel",
    "odigos-operator-rhel",
    "odigos-agents-rhel",
    "odigos-cli-rhel",
  ]
}

group "stack-rhel" {
  targets = [
    "odigos-ui-rhel",
    "odigos-collector-rhel",
    "odigos-odiglet-rhel",
    "odigos-autoscaler-rhel",
    "odigos-scheduler-rhel",
    "odigos-instrumentor-rhel",
    "odigos-agents-rhel",
    "odigos-cli-rhel",
  ]
}

target "_module_common" {
  context = "."
  platforms = ["linux/amd64", "linux/arm64"]
  args = {
    ODIGOS_VERSION = "${ODIGOS_TAG}"
    VERSION = "${ODIGOS_TAG}"
    RELEASE = "${ODIGOS_TAG}"
    LD_FLAGS = "-s -w"
  }
}

target "odigos-autoscaler" {
  inherits = ["_module_common"]
  dockerfile = "Dockerfile"
  args = {
    SERVICE_NAME = "autoscaler"
    SUMMARY = "Autoscaler for Odigos"
    DESCRIPTION = "Autoscaler manages the installation of Odigos components."
  }
  tags = [
    "${GCP_REPO}/odigos-autoscaler${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DEPOT_REPO}/odigos-autoscaler${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DOCKERHUB_REPO}/odigos-autoscaler${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${GHCR_REPO}/odigos-autoscaler${IMAGE_SUFFIX}:${ODIGOS_TAG}",
  ]
}

target "odigos-scheduler" {
  inherits = ["_module_common"]
  dockerfile = "Dockerfile"
  args = {
    SERVICE_NAME = "scheduler"
    SUMMARY = "Scheduler for Odigos"
    DESCRIPTION = "Scheduler manages the installation of OpenTelemetry Collectors with Odigos."
  }
  tags = [
    "${GCP_REPO}/odigos-scheduler${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DEPOT_REPO}/odigos-scheduler${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DOCKERHUB_REPO}/odigos-scheduler${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${GHCR_REPO}/odigos-scheduler${IMAGE_SUFFIX}:${ODIGOS_TAG}",
  ]
}

target "odigos-instrumentor" {
  inherits = ["_module_common"]
  dockerfile = "Dockerfile"
  args = {
    SERVICE_NAME = "instrumentor"
    SUMMARY = "Instrumentor for Odigos"
    DESCRIPTION = "Instrumentor manages auto-instrumentation for workloads with Odigos."
  }
  tags = [
    "${GCP_REPO}/odigos-instrumentor${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DEPOT_REPO}/odigos-instrumentor${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DOCKERHUB_REPO}/odigos-instrumentor${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${GHCR_REPO}/odigos-instrumentor${IMAGE_SUFFIX}:${ODIGOS_TAG}",
  ]
}

target "odigos-collector" {
  inherits = ["_module_common"]
  dockerfile = "collector/Dockerfile"
  args = {
    SERVICE_NAME = "collector"
    SUMMARY = "Odigos Collector"
    DESCRIPTION = "The Odigos build of the OpenTelemetry Collector."
  }
  tags = [
    "${GCP_REPO}/odigos-collector${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DEPOT_REPO}/odigos-collector${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DOCKERHUB_REPO}/odigos-collector${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${GHCR_REPO}/odigos-collector${IMAGE_SUFFIX}:${ODIGOS_TAG}",
  ]
}

target "odigos-odiglet" {
  inherits = ["_module_common"]
  dockerfile = "odiglet/Dockerfile"
  args = {
    SERVICE_NAME = "odiglet"
    SUMMARY = "Odiglet for Odigos"
    DESCRIPTION = "Odiglet is the core component of Odigos managing auto-instrumentation. This container requires a root user to run and manage eBPF programs."
  }
  tags = [
    "${GCP_REPO}/odigos-odiglet${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DEPOT_REPO}/odigos-odiglet${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DOCKERHUB_REPO}/odigos-odiglet${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${GHCR_REPO}/odigos-odiglet${IMAGE_SUFFIX}:${ODIGOS_TAG}",
  ]
}

target "odigos-ui" {
  inherits = ["_module_common"]
  dockerfile = "frontend/Dockerfile"
  args = {
    SERVICE_NAME = "ui"
    SUMMARY = "UI for Odigos"
    DESCRIPTION = "UI provides the frontend webapp for managing an Odigos installation."
  }
  tags = [
    "${GCP_REPO}/odigos-ui${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DEPOT_REPO}/odigos-ui${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DOCKERHUB_REPO}/odigos-ui${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${GHCR_REPO}/odigos-ui${IMAGE_SUFFIX}:${ODIGOS_TAG}",
  ]
}

target "odigos-operator" {
  inherits = ["_module_common"]
  dockerfile = "operator/Dockerfile"
  args = {
    SERVICE_NAME = "operator"
    SUMMARY = "Odigos Operator"
    DESCRIPTION = "Kubernetes Operator for Odigos installs Odigos"
  }
  tags = [
    "${GCP_REPO}/odigos-operator${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DEPOT_REPO}/odigos-operator${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DOCKERHUB_REPO}/odigos-operator${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${GHCR_REPO}/odigos-operator${IMAGE_SUFFIX}:${ODIGOS_TAG}",
  ]
}

target "odigos-agents" {
  inherits = ["_module_common"]
  dockerfile = "odiglet/Dockerfile"
  target = "agents"
  args = {
    SERVICE_NAME = "agents"
    SUMMARY = "Init container for Odigos"
    DESCRIPTION = "Init container for Odigos managing auto-instrumentation. This container requires a root user to run and manage eBPF programs."
  }
  tags = [
    "${GCP_REPO}/odigos-agents${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DEPOT_REPO}/odigos-agents${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DOCKERHUB_REPO}/odigos-agents${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${GHCR_REPO}/odigos-agents${IMAGE_SUFFIX}:${ODIGOS_TAG}",
  ]
}

target "odigos-cli" {
  context = "."
  platforms = ["linux/amd64", "linux/arm64"]
  dockerfile = "cli/Dockerfile"
  args = {
    ODIGOS_TAG = "${ODIGOS_TAG}"
    VERSION = "${ODIGOS_TAG}"
    RELEASE = "${ODIGOS_TAG}"
    SUMMARY = "Odigos CLI"
    DESCRIPTION = "Odigos CLI to install and manage Odigos in your Kubernetes cluster."
  }
  tags = [
    "${GCP_REPO}/odigos-cli${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DEPOT_REPO}/odigos-cli${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${DOCKERHUB_REPO}/odigos-cli${IMAGE_SUFFIX}:${ODIGOS_TAG}",
    "${GHCR_REPO}/odigos-cli${IMAGE_SUFFIX}:${ODIGOS_TAG}",
  ]
}

target "odigos-autoscaler-rhel" {
  inherits = ["odigos-autoscaler"]
  target = "rhel"
  args = {
    RHEL = "true"
  }
}

target "odigos-scheduler-rhel" {
  inherits = ["odigos-scheduler"]
  target = "rhel"
  args = {
    RHEL = "true"
  }
}

target "odigos-instrumentor-rhel" {
  inherits = ["odigos-instrumentor"]
  target = "rhel"
  args = {
    RHEL = "true"
  }
}

target "odigos-collector-rhel" {
  inherits = ["odigos-collector"]
  target = "rhel"
  args = {
    RHEL = "true"
  }
}

target "odigos-odiglet-rhel" {
  inherits = ["odigos-odiglet"]
  target = "rhel"
  args = {
    RHEL = "true"
  }
}

target "odigos-ui-rhel" {
  inherits = ["odigos-ui"]
  target = "rhel"
  args = {
    RHEL = "true"
  }
}

target "odigos-operator-rhel" {
  inherits = ["odigos-operator"]
  target = "rhel"
  args = {
    RHEL = "true"
  }
}

target "odigos-agents-rhel" {
  inherits = ["odigos-agents"]
  target = "agents-rhel"
  args = {
    RHEL = "true"
  }
}

target "odigos-cli-rhel" {
  inherits = ["odigos-cli"]
  target = "rhel"
}
