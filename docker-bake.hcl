# ──────────────────────────────────────────────
# docker-bake.hcl  (compatible with Buildx <0.11)
# ──────────────────────────────────────────────

variable "TAG"        { default = "latest" }
variable "ORG"        { default = "registry.odigos.io" }
variable "IMG_SUFFIX" { default = "" }
variable "SHORT_COMMIT" { default = "" }
variable "DATE"         { default = "" }

# ── Shared metadata (was “template common”) ───
target "common" {
  args = {
    ODIGOS_VERSION = "${TAG}"
    VERSION        = "${TAG}"
    RELEASE        = "${TAG}"
  }
}

# ── RHEL / UBI-9 metadata (was “template rhel”) ──
target "rhel" {
  inherits = ["common"]
  args = { IMG_SUFFIX = "-ubi9" }
}

# ── Main images ───────────────────────────────
target "operator" {
  inherits   = ["common"]
  context    = "."
  dockerfile = "operator/Dockerfile"
  args = {
    SERVICE_NAME = "operator"
    SUMMARY      = "Odigos Operator"
    DESCRIPTION  = "Kubernetes Operator for Odigos installs Odigos"
  }
  tags = ["${ORG}/odigos-operator${IMG_SUFFIX}:${TAG}"]
}

target "odiglet" {
  inherits   = ["common"]
  context    = "."
  dockerfile = "odiglet/Dockerfile"
  args = {
    SERVICE_NAME = "odiglet"
    SUMMARY      = "Odiglet for Odigos"
    DESCRIPTION  = "Odiglet is the core component of Odigos managing auto-instrumentation."
  }
  tags = ["${ORG}/odigos-odiglet${IMG_SUFFIX}:${TAG}"]
}

target "autoscaler" {
  inherits = ["common"]
  context  = "."
  args = {
    SERVICE_NAME = "autoscaler"
    SUMMARY      = "Autoscaler for Odigos"
    DESCRIPTION  = "Autoscaler manages the installation of Odigos components."
  }
  tags = ["${ORG}/odigos-autoscaler${IMG_SUFFIX}:${TAG}"]
}

target "instrumentor" {
  inherits = ["common"]
  context  = "."
  args = {
    SERVICE_NAME = "instrumentor"
    SUMMARY      = "Instrumentor for Odigos"
    DESCRIPTION  = "Instrumentor manages auto-instrumentation for workloads with Odigos."
  }
  tags = ["${ORG}/odigos-instrumentor${IMG_SUFFIX}:${TAG}"]
}

target "scheduler" {
  inherits = ["common"]
  context  = "."
  args = {
    SERVICE_NAME = "scheduler"
    SUMMARY      = "Scheduler for Odigos"
    DESCRIPTION  = "Scheduler manages the installation of OpenTelemetry Collectors with Odigos."
  }
  tags = ["${ORG}/odigos-scheduler${IMG_SUFFIX}:${TAG}"]
}

target "collector" {
  inherits   = ["common"]
  context    = "collector"
  dockerfile = "Dockerfile"
  args = {
    SERVICE_NAME = "collector"
    SUMMARY      = "Odigos Collector"
    DESCRIPTION  = "The Odigos build of the OpenTelemetry Collector."
  }
  tags = ["${ORG}/odigos-collector${IMG_SUFFIX}:${TAG}"]
}

target "ui" {
  inherits   = ["common"]
  context    = "."
  dockerfile = "frontend/Dockerfile"
  args = {
    SERVICE_NAME = "ui"
    SUMMARY      = "UI for Odigos"
    DESCRIPTION  = "UI provides the frontend webapp for managing an Odigos installation."
  }
  tags = ["${ORG}/odigos-ui${IMG_SUFFIX}:${TAG}"]
}

target "cli" {
  inherits = ["common"]
  context  = "."
  dockerfile = "cli/Dockerfile"
  tags = ["${ORG}/odigos-cli${IMG_SUFFIX}:${TAG}"]
}

# ── UBI-9 / RHEL image variants ───────────────
target "operator-rhel" {
  inherits   = ["operator", "rhel"]
  dockerfile = "operator/Dockerfile.rhel"
}

target "odiglet-rhel" {
  inherits   = ["odiglet", "rhel"]
  dockerfile = "odiglet/Dockerfile.rhel"
}

target "autoscaler-rhel"    { inherits = ["autoscaler",   "rhel"] }
target "instrumentor-rhel"  { inherits = ["instrumentor", "rhel"] }
target "scheduler-rhel"     { inherits = ["scheduler",    "rhel"] }

target "collector-rhel" {
  inherits   = ["collector", "rhel"]
  dockerfile = "Dockerfile.rhel"
}

target "ui-rhel" {
  inherits   = ["ui", "rhel"]
  dockerfile = "frontend/Dockerfile.rhel"
}

target "cli-rhel" {
  inherits   = ["cli", "rhel"]
}


# ── Convenience groups ───────────────────────
group "images" {
  targets = [
    "operator","odiglet","autoscaler",
    "instrumentor","scheduler","collector","ui",
    "cli"
  ]
}

group "images-rhel" {
  targets = [
    "operator-rhel","odiglet-rhel","autoscaler-rhel",
    "instrumentor-rhel","scheduler-rhel",
    "collector-rhel","ui-rhel",
    "cli-rhel"
  ]
}
