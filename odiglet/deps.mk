
.PHONY: upgrade-odiglet-agent-version
upgrade-odiglet-agent-version:
	@if [ -z "$(AGENT_DISTRO)" ] || [ -z "$(AGENT_VERSION)" ]; then \
		echo 'ERROR: AGENT_DISTRO and AGENT_VERSION are required. Example: make upgrade-odiglet-agent-version AGENT_DISTRO=nodejs-community AGENT_VERSION=v0.0.9'; \
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
.PHONY: upgrade-odiglet-python-community-version
upgrade-odiglet-python-community-version:
	@if [ -z "$(AGENT_VERSION)" ]; then \
		echo 'ERROR: AGENT_VERSION is required. Example: make upgrade-odiglet-python-community-version AGENT_VERSION=v0.1.66'; \
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
