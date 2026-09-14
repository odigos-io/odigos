# Docker Bake definition for the OSS Odigos components.
#
# Single source of truth for how every OSS Docker image in this repo is built:
# which Dockerfile/stage, build args, tags, and the summary/description labels
# baked into the RHEL images. The Makefile (build-%, push-%, build-images, ...)
# is a thin wrapper around `docker buildx bake` using this file.
#
# Every component has two variants:
#   <name>       - the regular (distroless-based) image
#   <name>-rhel  - the RHEL/UBI-based, Red Hat certified variant (RHEL=true,
#                  built from the Dockerfile's "rhel" stage - "agents-rhel"
#                  for agents - tagged with the "-rhel-certified" suffix)
#
# Groups:
#   images / images-rhel - the 7 components deployed to a dev/e2e cluster
#                           (matches `make build-images` / `build-images-rhel`)
#   default (default group), rhel - aliases for images / images-rhel
#   oss / oss-rhel        - every OSS component, including operator and cli
#   all                   - everything: oss + oss-rhel
#
# Usage:
#   docker buildx bake                                   # images group, linux/amd64
#   docker buildx bake ui collector                      # a subset
#   docker buildx bake oss                                # every regular OSS image
#   docker buildx bake rhel                              # images-rhel
#   docker buildx bake all                                # everything, regular + RHEL
#   docker buildx bake --set *.platform=linux/arm64       # arm64 only
#   PLATFORMS=linux/amd64,linux/arm64 docker buildx bake  # multi-arch (needs --push to publish a manifest list)
#   TAG=v1.2.3 ORG=docker.io/keyval docker buildx bake --push

variable "TAG" {
  default = "latest"
}

# Registry/org prefix. Defaults to the OSS org used by `make build-images`
# (OSS_ORG in Makefile). Set to registry.odigos.io for enterprise builds.
variable "ORG" {
  default = "docker.io/keyval"
}

# Appended to every *regular* image name, e.g. for a custom fork/staging org.
# RHEL images always use the fixed "-rhel-certified" suffix instead (see below).
variable "IMG_SUFFIX" {
  default = ""
}

# Comma-separated platform list, e.g. "linux/amd64,linux/arm64". A single
# entry produces a normal image; more than one requires --push (or a
# containerd image store) since a multi-platform result can't be `--load`ed.
variable "PLATFORMS" {
  default = "linux/amd64"
}

variable "LD_FLAGS" {
  default = ""
}

# Only consumed by the cli/cli-rhel targets (embedded in the --version output).
variable "SHORT_COMMIT" {
  default = ""
}

variable "DATE" {
  default = ""
}

# Chart version embedded in the CLI (must be SemVer for helm package; the
# Dockerfile normalizes a non-SemVer TAG, e.g. "e2e-test", to "0.0.0-<tag>").
# Defaults to TAG; override independently when the two need to differ.
variable "CHART_VERSION" {
  default = TAG
}

# Fixed suffix for RHEL/certified images, matching the historical Makefile
# behavior (RHEL=true always resolves to "-rhel-certified", replacing any
# custom IMG_SUFFIX).
RHEL_SUFFIX = "-rhel-certified"

# The 7 components historically built by `make build-images` (ORG's daemonset/
# deployment images loaded onto a dev/e2e cluster). Deliberately excludes
# operator and cli: neither is deployed by `make deploy`/e2e clusters, and
# operator's Dockerfile requires ODIGOS_VERSION to be valid SemVer (it doesn't
# normalize a tag like "e2e-test" the way the other Dockerfiles do), which
# breaks the e2e/dev-cluster flows (TAG=e2e-test) if it's pulled in here.
group "images" {
  targets = [
    "autoscaler",
    "scheduler",
    "instrumentor",
    "odiglet",
    "agents",
    "collector",
    "ui",
  ]
}

group "images-rhel" {
  targets = [
    "autoscaler-rhel",
    "scheduler-rhel",
    "instrumentor-rhel",
    "odiglet-rhel",
    "agents-rhel",
    "collector-rhel",
    "ui-rhel",
  ]
}

# Default target of a bare `docker buildx bake` - matches `make build-images`.
group "default" {
  targets = ["images"]
}

# Matches `make build-images-rhel` / `push-images-rhel`.
group "rhel" {
  targets = ["images-rhel"]
}

# Every OSS component, including operator and cli (not part of build-images).
group "oss" {
  targets = ["images", "operator", "cli"]
}

group "oss-rhel" {
  targets = ["images-rhel", "operator-rhel", "cli-rhel"]
}

# Truly everything: every component, regular and RHEL.
group "all" {
  targets = ["oss", "oss-rhel"]
}

target "_common" {
  context   = "."
  platforms = split(",", PLATFORMS)
  args = {
    VERSION  = TAG
    RELEASE  = TAG
    LD_FLAGS = LD_FLAGS
    RHEL     = "false"
  }
}

# Shared by every "-rhel" target: builds the Dockerfile's RHEL/UBI stage.
target "_rhel" {
  inherits = ["_common"]
  args = {
    RHEL = "true"
  }
}

# autoscaler, scheduler and instrumentor all share the root Dockerfile,
# which builds whichever Go service is named via SERVICE_NAME.
target "_service" {
  inherits   = ["_common"]
  dockerfile = "Dockerfile"
}

target "_service-rhel" {
  inherits   = ["_rhel"]
  dockerfile = "Dockerfile"
  target     = "rhel"
}

target "autoscaler" {
  inherits = ["_service"]
  tags     = ["${ORG}/odigos-autoscaler${IMG_SUFFIX}:${TAG}"]
  args = {
    SERVICE_NAME   = "autoscaler"
    ODIGOS_VERSION = TAG
    SUMMARY        = "Autoscaler for Odigos"
    DESCRIPTION    = "Autoscaler manages the installation of Odigos components."
  }
}

target "autoscaler-rhel" {
  inherits = ["autoscaler", "_service-rhel"]
  tags     = ["${ORG}/odigos-autoscaler${RHEL_SUFFIX}:${TAG}"]
}

target "scheduler" {
  inherits = ["_service"]
  tags     = ["${ORG}/odigos-scheduler${IMG_SUFFIX}:${TAG}"]
  args = {
    SERVICE_NAME   = "scheduler"
    ODIGOS_VERSION = TAG
    SUMMARY        = "Scheduler for Odigos"
    DESCRIPTION    = "Scheduler manages the installation of OpenTelemetry Collectors with Odigos."
  }
}

target "scheduler-rhel" {
  inherits = ["scheduler", "_service-rhel"]
  tags     = ["${ORG}/odigos-scheduler${RHEL_SUFFIX}:${TAG}"]
}

target "instrumentor" {
  inherits = ["_service"]
  tags     = ["${ORG}/odigos-instrumentor${IMG_SUFFIX}:${TAG}"]
  args = {
    SERVICE_NAME   = "instrumentor"
    ODIGOS_VERSION = TAG
    SUMMARY        = "Instrumentor for Odigos"
    DESCRIPTION    = "Instrumentor manages auto-instrumentation for workloads with Odigos."
  }
}

target "instrumentor-rhel" {
  inherits = ["instrumentor", "_service-rhel"]
  tags     = ["${ORG}/odigos-instrumentor${RHEL_SUFFIX}:${TAG}"]
}

target "odiglet" {
  inherits   = ["_common"]
  dockerfile = "odiglet/Dockerfile"
  tags       = ["${ORG}/odigos-odiglet${IMG_SUFFIX}:${TAG}"]
  args = {
    ODIGOS_VERSION = TAG
    SUMMARY        = "Odiglet for Odigos"
    DESCRIPTION    = "Odiglet is the core component of Odigos managing auto-instrumentation. This container requires a root user to run and manage eBPF programs."
  }
}

target "odiglet-rhel" {
  inherits   = ["odiglet", "_rhel"]
  dockerfile = "odiglet/Dockerfile"
  target     = "rhel"
  tags       = ["${ORG}/odigos-odiglet${RHEL_SUFFIX}:${TAG}"]
}

# Init container image: same Dockerfile as odiglet, but stops at the "agents" stage.
target "agents" {
  inherits   = ["_common"]
  dockerfile = "odiglet/Dockerfile"
  target     = "agents"
  tags       = ["${ORG}/odigos-agents${IMG_SUFFIX}:${TAG}"]
  args = {
    ODIGOS_VERSION = TAG
    SUMMARY        = "Init container for Odigos"
    DESCRIPTION    = "Init container for Odigos managing auto-instrumentation. This container requires a root user to run and manage eBPF programs."
  }
}

target "agents-rhel" {
  inherits   = ["agents", "_rhel"]
  dockerfile = "odiglet/Dockerfile"
  target     = "agents-rhel"
  tags       = ["${ORG}/odigos-agents${RHEL_SUFFIX}:${TAG}"]
}

target "collector" {
  inherits   = ["_common"]
  dockerfile = "collector/Dockerfile"
  tags       = ["${ORG}/odigos-collector${IMG_SUFFIX}:${TAG}"]
  args = {
    SUMMARY     = "Odigos Collector"
    DESCRIPTION = "The Odigos build of the OpenTelemetry Collector."
  }
}

target "collector-rhel" {
  inherits   = ["collector", "_rhel"]
  dockerfile = "collector/Dockerfile"
  target     = "rhel"
  tags       = ["${ORG}/odigos-collector${RHEL_SUFFIX}:${TAG}"]
}

target "ui" {
  inherits   = ["_common"]
  dockerfile = "frontend/Dockerfile"
  tags       = ["${ORG}/odigos-ui${IMG_SUFFIX}:${TAG}"]
  args = {
    SUMMARY     = "UI for Odigos"
    DESCRIPTION = "UI provides the frontend webapp for managing an Odigos installation."
  }
}

target "ui-rhel" {
  inherits   = ["ui", "_rhel"]
  dockerfile = "frontend/Dockerfile"
  target     = "rhel"
  tags       = ["${ORG}/odigos-ui${RHEL_SUFFIX}:${TAG}"]
}

target "operator" {
  inherits   = ["_common"]
  dockerfile = "operator/Dockerfile"
  tags       = ["${ORG}/odigos-operator${IMG_SUFFIX}:${TAG}"]
  args = {
    ODIGOS_VERSION = TAG
    SUMMARY        = "Odigos Operator"
    DESCRIPTION    = "Kubernetes Operator for Odigos installs Odigos"
  }
}

target "operator-rhel" {
  inherits   = ["operator", "_rhel"]
  dockerfile = "operator/Dockerfile"
  target     = "rhel"
  tags       = ["${ORG}/odigos-operator${RHEL_SUFFIX}:${TAG}"]
}

target "cli" {
  inherits   = ["_common"]
  dockerfile = "cli/Dockerfile"
  tags       = ["${ORG}/odigos-cli${IMG_SUFFIX}:${TAG}"]
  args = {
    CHART_VERSION = CHART_VERSION
    SUMMARY       = "Odigos CLI"
    DESCRIPTION   = "Odigos CLI to install and manage Odigos in your Kubernetes cluster."
    SHORT_COMMIT  = SHORT_COMMIT
    DATE          = DATE
  }
}

target "cli-rhel" {
  inherits   = ["cli", "_rhel"]
  dockerfile = "cli/Dockerfile"
  target     = "rhel"
  tags       = ["${ORG}/odigos-cli${RHEL_SUFFIX}:${TAG}"]
}
