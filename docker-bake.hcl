# Bake all OSS components from the odigos repo root.
#
#   docker buildx bake oss          # charts + controllers + odiglet + collector + ui
#   docker buildx bake oss-all      # oss + cli
#   docker buildx bake e2e          # same set as `make build-images` (for CI)
#   docker buildx bake instrumentor # single target
#
# Odigelet-only bake (same targets): odiglet/docker/docker-bake.hcl
# BUILDKIT_CACHE_MOUNT_NS isolates Dockerfile --mount=type=cache per target on Depot.

variable "IMAGE_TAG" {
  default = "dev"
}

variable "PLATFORM" {
  default = "linux/arm64"
}

# Repo root when baking from odigos/. Override if bake cwd differs.
variable "ODIGOS_OSS_REPO_DIR" {
  default = "."
}

variable "CHART_VERSION" {
  default = "dev"
}

variable "SHORT_COMMIT" {
  default = "dev"
}

variable "DATE" {
  default = "1970-01-01T00:00:00Z"
}

variable "OPENTELEMETRY_NODE_REPO_DIR" {
  default = "../opentelemetry-node"
}

variable "ODIGLET_DOCKER_DIR" {
  default = "odiglet/docker"
}

variable "ODIGLET_BUILDER_DOCKERFILE" {
  default = "odiglet/docker/Dockerfile.builder"
}

variable "ODIGLET_COMMON_BINARIES_DOCKERFILE" {
  default = "odiglet/docker/Dockerfile.common-binaries"
}

variable "AGENTS_REGISTRY" {
  default = "public.ecr.aws/odigos/agents"
}

variable "NODEJS_COMMUNITY_VERSION" {}

variable "NODEJS_COMMUNITY_14_VERSION" {}

variable "PHP_COMMUNITY_VERSION" {}

variable "RUBY_COMMUNITY_VERSION" {}

variable "NODEJS_COMMUNITY_IMAGE" {
  default = "docker-image://${AGENTS_REGISTRY}/nodejs-community:${NODEJS_COMMUNITY_VERSION}"
}

variable "NODEJS_COMMUNITY_14_IMAGE" {
  default = "docker-image://${AGENTS_REGISTRY}/nodejs-community-14:${NODEJS_COMMUNITY_14_VERSION}"
}

variable "PHP_COMMUNITY_IMAGE" {
  default = "docker-image://${AGENTS_REGISTRY}/php-community:${PHP_COMMUNITY_VERSION}"
}

variable "RUBY_COMMUNITY_IMAGE" {
  default = "docker-image://${AGENTS_REGISTRY}/ruby-community:${RUBY_COMMUNITY_VERSION}"
}

group "default" {
  targets = ["oss"]
}

group "oss" {
  targets = ["charts", "instrumentor", "autoscaler", "scheduler", "odiglet", "collector", "ui"]
}

group "oss-all" {
  targets = ["charts", "instrumentor", "autoscaler", "scheduler", "odiglet", "collector", "ui", "cli"]
}

# Same set as `make build-images` used by e2e (no charts / cli).
group "e2e" {
  targets = ["instrumentor", "autoscaler", "scheduler", "odiglet", "collector", "ui", "agents"]
}

######### controllers / collector / ui / charts / cli #########

target "instrumentor" {
  context    = ODIGOS_OSS_REPO_DIR
  dockerfile = "Dockerfile"
  platforms  = [PLATFORM]
  args = {
    SERVICE_NAME            = "instrumentor"
    BUILDKIT_CACHE_MOUNT_NS = "instrumentor"
  }
  provenance = false
}

target "autoscaler" {
  context    = ODIGOS_OSS_REPO_DIR
  dockerfile = "Dockerfile"
  platforms  = [PLATFORM]
  args = {
    SERVICE_NAME            = "autoscaler"
    BUILDKIT_CACHE_MOUNT_NS = "autoscaler"
  }
  provenance = false
}

target "scheduler" {
  context    = ODIGOS_OSS_REPO_DIR
  dockerfile = "Dockerfile"
  platforms  = [PLATFORM]
  args = {
    SERVICE_NAME            = "scheduler"
    BUILDKIT_CACHE_MOUNT_NS = "scheduler"
  }
  provenance = false
}

target "collector" {
  context    = ODIGOS_OSS_REPO_DIR
  dockerfile = "collector/fast.Dockerfile"
  platforms  = [PLATFORM]
  args = {
    BUILDKIT_CACHE_MOUNT_NS = "collector"
  }
  provenance = false
}

target "ui" {
  context    = ODIGOS_OSS_REPO_DIR
  dockerfile = "frontend/Dockerfile"
  platforms  = [PLATFORM]
  args = {
    BUILDKIT_CACHE_MOUNT_NS = "ui"
  }
  provenance = false
}

target "charts" {
  context    = ODIGOS_OSS_REPO_DIR
  dockerfile = "helm/helm-charts.Dockerfile"
  platforms  = [PLATFORM]
  args = {
    CHART_VERSION = CHART_VERSION
  }
  provenance = false
}

target "cli" {
  context    = ODIGOS_OSS_REPO_DIR
  dockerfile = "cli/Dockerfile"
  platforms  = [PLATFORM]
  args = {
    VERSION                 = IMAGE_TAG
    CHART_VERSION           = CHART_VERSION
    SHORT_COMMIT            = SHORT_COMMIT
    DATE                    = DATE
    BUILDKIT_CACHE_MOUNT_NS = "cli"
  }
  provenance = false
}

######### odiglet / agents (Dockerfiles in odiglet/docker/) #########
# Intermediate targets use cacheonly + no platforms so Bake builds them once as
# named contexts (avoids dual solve / "another build is calculating").

target "odiglet-dotnet" {
  context    = ODIGLET_DOCKER_DIR
  dockerfile = "Dockerfile.dotnet"
  output     = [{ type = "cacheonly" }]
  provenance = false
}

target "odiglet-agents-builder" {
  context    = ODIGLET_DOCKER_DIR
  dockerfile = "Dockerfile.agents-builder"
  contexts = {
    dotnet-builder      = "target:odiglet-dotnet"
    php-community       = PHP_COMMUNITY_IMAGE
    ruby-community      = RUBY_COMMUNITY_IMAGE
    nodejs-community    = NODEJS_COMMUNITY_IMAGE
    nodejs-community-14 = NODEJS_COMMUNITY_14_IMAGE
  }
  output     = [{ type = "cacheonly" }]
  provenance = false
}

target "agents" {
  context    = ODIGLET_DOCKER_DIR
  dockerfile = "Dockerfile.agents"
  platforms  = [PLATFORM]
  contexts = {
    agents-builder = "target:odiglet-agents-builder"
  }
  provenance = false
}

target "agents-rhel" {
  context    = ODIGLET_DOCKER_DIR
  dockerfile = "Dockerfile.agents-rhel"
  platforms  = [PLATFORM]
  contexts = {
    agents-builder = "target:odiglet-agents-builder"
  }
  provenance = false
}

target "odiglet-builder" {
  context    = ODIGOS_OSS_REPO_DIR
  dockerfile = ODIGLET_BUILDER_DOCKERFILE
  args = {
    BUILDKIT_CACHE_MOUNT_NS = "odiglet-builder"
  }
  output     = [{ type = "cacheonly" }]
  provenance = false
}

target "odiglet-builder-rhel" {
  context    = ODIGOS_OSS_REPO_DIR
  dockerfile = ODIGLET_BUILDER_DOCKERFILE
  args = {
    BUILDKIT_CACHE_MOUNT_NS = "odiglet-builder-rhel"
    RHEL                    = "true"
  }
  output     = [{ type = "cacheonly" }]
  provenance = false
}

target "odiglet-common-binaries" {
  context    = ODIGOS_OSS_REPO_DIR
  dockerfile = ODIGLET_COMMON_BINARIES_DOCKERFILE
  args = {
    BUILDKIT_CACHE_MOUNT_NS = "odiglet-common-binaries"
  }
  output     = [{ type = "cacheonly" }]
  provenance = false
}

target "odiglet" {
  context    = ODIGLET_DOCKER_DIR
  dockerfile = "Dockerfile"
  platforms  = [PLATFORM]
  contexts = {
    builder         = "target:odiglet-builder"
    common-binaries = "target:odiglet-common-binaries"
    agents-builder  = "target:odiglet-agents-builder"
  }
  provenance = false
}

target "nodejs-community-local" {
  context    = OPENTELEMETRY_NODE_REPO_DIR
  dockerfile = "Dockerfile"
  args = {
    AGENT_VERSION = IMAGE_TAG
  }
  output     = [{ type = "cacheonly" }]
  provenance = false
}

target "odiglet-agents-builder-nodejs-dev" {
  context    = ODIGLET_DOCKER_DIR
  dockerfile = "Dockerfile.agents-builder"
  contexts = {
    dotnet-builder      = "target:odiglet-dotnet"
    php-community       = PHP_COMMUNITY_IMAGE
    ruby-community      = RUBY_COMMUNITY_IMAGE
    nodejs-community    = "target:nodejs-community-local"
    nodejs-community-14 = NODEJS_COMMUNITY_14_IMAGE
  }
  output     = [{ type = "cacheonly" }]
  provenance = false
}

target "odiglet-nodejs-dev" {
  context    = ODIGLET_DOCKER_DIR
  dockerfile = "Dockerfile"
  platforms  = [PLATFORM]
  contexts = {
    builder         = "target:odiglet-builder"
    common-binaries = "target:odiglet-common-binaries"
    agents-builder  = "target:odiglet-agents-builder-nodejs-dev"
  }
  provenance = false
}

target "odiglet-builder-debug" {
  context    = ODIGLET_DOCKER_DIR
  dockerfile = "Dockerfile.builder-debug"
  contexts = {
    builder = "target:odiglet-builder"
  }
  output     = [{ type = "cacheonly" }]
  provenance = false
}

target "odiglet-debug" {
  context    = ODIGLET_DOCKER_DIR
  dockerfile = "Dockerfile.debug"
  platforms  = [PLATFORM]
  contexts = {
    builder        = "target:odiglet-builder-debug"
    agents-builder = "target:odiglet-agents-builder"
  }
  provenance = false
}

target "odiglet-rhel" {
  context    = ODIGLET_DOCKER_DIR
  dockerfile = "Dockerfile.rhel"
  platforms  = [PLATFORM]
  contexts = {
    builder         = "target:odiglet-builder-rhel"
    common-binaries = "target:odiglet-common-binaries"
    agents-builder  = "target:odiglet-agents-builder"
  }
  provenance = false
}
