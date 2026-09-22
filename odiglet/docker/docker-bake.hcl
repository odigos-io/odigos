# OSS odiglet bake targets (split Dockerfiles in this directory).
# Prefer the repo-root docker-bake.hcl for full OSS builds:
#   docker buildx bake oss
#
# This file is for odiglet-only builds from repo root:
#   docker buildx bake -f odiglet/docker/docker-bake.hcl odiglet
#
# Intermediate targets use cacheonly + no platforms so Bake builds them once as
# named contexts (avoids dual solve / "another build is calculating").

variable "IMAGE_TAG" {
  default = "dev"
}

variable "PLATFORM" {
  default = "linux/arm64"
}

variable "ODIGOS_OSS_REPO_DIR" {
  default = "."
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
