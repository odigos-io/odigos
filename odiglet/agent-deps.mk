# Instrumentation-agent version bumps. Kept out of the main Makefile so the agent-update CI
# (ci-core/apply-agent-version-update) can drive it via `make -f agent-deps.mk upgrade-agent` without touching
# the build targets. Run targets here with `make -f agent-deps.mk <target>`.

# Path to THIS makefile, so the dispatcher can recurse into its own sub-targets.
MK := $(lastword $(MAKEFILE_LIST))

# Generic entrypoint used by CI. Maps a canonical INSTRUMENTATION_AGENT to the concrete target for this
# repo. Unknown categories are a no-op so a release can safely broadcast to all consumers.
.PHONY: upgrade-agent
upgrade-agent:
	@if [ -z "$(INSTRUMENTATION_AGENT)" ] || [ -z "$(AGENT_VERSION)" ]; then \
		echo 'ERROR: INSTRUMENTATION_AGENT and AGENT_VERSION are required. Example: make -f agent-deps.mk upgrade-agent INSTRUMENTATION_AGENT=php AGENT_VERSION=v0.3.2'; \
		exit 1; \
	fi
	@category=$$(echo "$(INSTRUMENTATION_AGENT)" | tr '[:upper:]' '[:lower:]'); \
	case "$$category" in \
		php|php-community) \
			$(MAKE) -f $(MK) upgrade-image-agent-version AGENT_DISTRO=php-community AGENT_VERSION=$(AGENT_VERSION) ;; \
		ruby|ruby-community) \
			$(MAKE) -f $(MK) upgrade-image-agent-version AGENT_DISTRO=ruby-community AGENT_VERSION=$(AGENT_VERSION) ;; \
		nodejs|nodejs-community) \
			$(MAKE) -f $(MK) upgrade-image-agent-version AGENT_DISTRO=nodejs-community AGENT_VERSION=$(AGENT_VERSION) ;; \
		python|python-community) \
			$(MAKE) -f $(MK) upgrade-python-community-version AGENT_VERSION=$(AGENT_VERSION) ;; \
		*) \
			echo "Instrumentation agent '$(INSTRUMENTATION_AGENT)' is not consumed by odigos OSS; nothing to do." ;; \
	esac

# Generic image-tag bump for image agents (php, ruby, nodejs-community, …). Updates both Dockerfile and
# debug.Dockerfile. AGENT_DISTRO matches the image name in the dockerfiles (e.g. php-community).
.PHONY: upgrade-image-agent-version
upgrade-image-agent-version:
	@if [ -z "$(AGENT_DISTRO)" ] || [ -z "$(AGENT_VERSION)" ]; then \
		echo 'ERROR: AGENT_DISTRO and AGENT_VERSION are required. Example: make -f agent-deps.mk upgrade-image-agent-version AGENT_DISTRO=nodejs-community AGENT_VERSION=v0.0.9'; \
		exit 1; \
	fi
	@echo "Updating $(AGENT_DISTRO) agent image tag to $(AGENT_VERSION) in Dockerfile and debug.Dockerfile"
	@if [ "$(shell uname)" = "Darwin" ]; then \
		sed -i '' -E 's|$(AGENT_DISTRO):[0-9A-Za-z._-]+|$(AGENT_DISTRO):$(AGENT_VERSION)|g' Dockerfile; \
		sed -i '' -E 's|$(AGENT_DISTRO):[0-9A-Za-z._-]+|$(AGENT_DISTRO):$(AGENT_VERSION)|g' debug.Dockerfile; \
	else \
		sed -i -E 's|$(AGENT_DISTRO):[0-9A-Za-z._-]+|$(AGENT_DISTRO):$(AGENT_VERSION)|g' Dockerfile; \
		sed -i -E 's|$(AGENT_DISTRO):[0-9A-Za-z._-]+|$(AGENT_DISTRO):$(AGENT_VERSION)|g' debug.Dockerfile; \
	fi

# python-community has two image tags pinned in the dockerfiles, this upgrades the specific agent version that needs to be updated and not python3.8
.PHONY: upgrade-python-community-version
upgrade-python-community-version:
	@if [ -z "$(AGENT_VERSION)" ]; then \
		echo 'ERROR: AGENT_VERSION is required. Example: make -f agent-deps.mk upgrade-python-community-version AGENT_VERSION=v0.1.66'; \
		exit 1; \
	fi
	@echo "Updating python-community agent image tag to $(AGENT_VERSION) in Dockerfile and debug.Dockerfile (preserving -py3.8 pin)"
	@if [ "$(shell uname)" = "Darwin" ]; then \
		sed -i '' -E 's|python-community:[0-9A-Za-z._-]+( +/instrumentations/python )|python-community:$(AGENT_VERSION)\1|g' Dockerfile; \
		sed -i '' -E 's|python-community:[0-9A-Za-z._-]+( +/instrumentations/python )|python-community:$(AGENT_VERSION)\1|g' debug.Dockerfile; \
	else \
		sed -i -E 's|python-community:[0-9A-Za-z._-]+( +/instrumentations/python )|python-community:$(AGENT_VERSION)\1|g' Dockerfile; \
		sed -i -E 's|python-community:[0-9A-Za-z._-]+( +/instrumentations/python )|python-community:$(AGENT_VERSION)\1|g' debug.Dockerfile; \
	fi
