
# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.23

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

TOOLS := $(CURDIR)/../.tools

.PHONY: all
all: generate manifests generate-client sync-helm-crd docgen

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: tools
tools:
	@if [ ! -d "$(TOOLS)" ]; then \
		mkdir $(TOOLS); \
	fi

.PHONY: genref
genref: tools
	GOBIN=$(TOOLS) go install github.com/kubernetes-sigs/reference-docs/genref@v0.28.0

.PHONY: docgen
docgen: genref
	@if [ ! -f "$(CURDIR)/../go.work" ]; then \
		echo "go.work not found; creating workspace (required for operator.odigos.io API docs)..."; \
		cd $(CURDIR)/.. && go work init && go work use -r .; \
	fi
	rm -rf $(CURDIR)/../docs/snippets/shared/api-reference/*
	$(TOOLS)/genref -c genref-config.yaml -f markdown -o $(CURDIR)/../docs/snippets/shared/api-reference
	for file in $(CURDIR)/../docs/snippets/shared/api-reference/*; do \
		mv $${file} $${file%.md}.mdx; \
	done
	# Fix known genref output issues (trailing whitespace + broken MDX <td> formatting)
	python3 $(CURDIR)/fix-genref-output.py $(CURDIR)/../docs/snippets/shared/api-reference

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	# remove the existing CRD files directory before recreating, so removed files will not be left over
	@rm -rf config/crd/bases
	# Regular Odigos CRDs (odigos.io + actions.odigos.io). The cloud connector CRD is intentionally
	# excluded here and generated separately below, so it is not bundled with the rest of the CRDs and
	# can be shipped only by the odigos-central Helm chart.
	$(CONTROLLER_GEN) crd:allowDangerousTypes=true webhook paths="./odigos/..." paths="./actions/..." output:crd:artifacts:config=config/crd/bases
	# Cloud connector CRD, kept out of config/crd/bases so sync-helm-crd only ships it to odigos-central.
	@rm -rf cloud-connectors/config/crd/bases
	$(CONTROLLER_GEN) crd:allowDangerousTypes=true paths="./cloud-connectors/..." output:crd:artifacts:config=cloud-connectors/config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..." paths="../common/..."

.PHONY: install
install: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	kubectl apply -f config/crd/bases

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.19.0
ENVTEST_VERSION ?= release-0.22

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_VERSION)

.PHONY: generate-client
generate-client:
	GOWORK=off go mod vendor
	./hack/update-codegen.sh
	rm -rf ./vendor

.PHONY: sync-helm
sync-helm-crd:
	@echo "Syncing regular CRD yamls to the odigos Helm chart"
	@rm ../helm/odigos/templates/crds/* # remove existing files so we don't get leftovers
	@cp -r config/crd/bases/* ../helm/odigos/templates/crds/
	@echo "Syncing cloud connector CRD only to the odigos-central chart (guarded by connectors.enabled)"
	@{ echo '{{- if (include "connectors.enabled" .) }}'; cat cloud-connectors/config/crd/bases/odigos.io_odigoscloudconnectors.yaml; echo '{{- end }}'; } > ../helm/odigos-central/templates/crds/odigoscloudconnectors.yaml
