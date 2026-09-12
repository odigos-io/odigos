# CRC (OpenShift Local) lifecycle.
#   make -f crc.mk            list targets
#   make -f crc.mk doctor     check prerequisites

PULL_SECRET ?= $(HOME)/pull-secret.txt
CTX         ?= crc-admin
MIRROR      := https://mirror.openshift.com/pub/openshift-v4/clients/crc/latest
THIS        := $(firstword $(MAKEFILE_LIST))

# crc does not put oc on PATH and each recipe line is its own shell.
OC_BIN   := $(HOME)/.crc/bin/oc/oc
SSH_KEY  := $(HOME)/.crc/machines/crc/id_ed25519
SSH_PORT ?= 2222

CHART         ?= ./helm/odigos
ODIGOS_NS     ?= odigos-system
TAG           ?= v1.37.0-rc1
ODIGLET_IMAGE ?=
HELM_ARGS     ?=

# openshift.enabled alone pins images to registry.connect.redhat.com, which needs
# Red Hat Connect credentials; imagePrefix redirects to one we can pull from.
# The chart names the odiglet differently once an on-prem secret exists.
ifdef ODIGOS_TOKEN
IMAGE_PREFIX ?= registry.odigos.io
ODIGLET_KEY  := enterprise-odiglet
TOKEN_FLAG   := --set onPremToken=$$ODIGOS_TOKEN
else
IMAGE_PREFIX ?= docker.io/keyval
ODIGLET_KEY  := odiglet
endif

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} \
	  /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5); next } \
	  /^[a-z0-9][a-z0-9-]*:.*##/ { printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }' $(THIS)
	@echo ""

##@ Prerequisites

.PHONY: doctor
doctor: ## Check prerequisites and report what is missing
	@ok=1; \
	for t in crc kubectl; do \
	  if command -v $$t >/dev/null; then echo "  ok      $$t"; \
	  else echo "  MISSING $$t"; ok=0; fi; done; \
	if [ -x "$(OC_BIN)" ]; then echo "  ok      oc"; \
	else echo "  MISSING oc - installed by 'make -f $(THIS) setup'"; ok=0; fi; \
	if [ -f $(PULL_SECRET) ]; then echo "  ok      pull secret"; \
	else echo "  MISSING pull secret at $(PULL_SECRET)"; ok=0; fi; \
	if crc status 2>/dev/null | grep -q "OpenShift: *Running"; then echo "  ok      cluster running"; \
	else echo "  note    cluster not running - 'make -f $(THIS) start'"; fi; \
	[ $$ok -eq 1 ] || { echo ""; echo "Fix the MISSING items above, then re-run."; exit 1; }

.PHONY: require-crc
require-crc:
	@command -v crc >/dev/null || { \
	  echo "crc is not installed."; \
	  echo ""; \
	  echo "  make -f $(THIS) download    # downloads + verifies, prints a sudo line"; \
	  echo ""; \
	  echo "The install needs sudo: the pkg sets the setuid bit on crc-admin-helper,"; \
	  echo "and installing by hand leaves it non-setuid and 'crc setup' fails oddly."; \
	  exit 1; }

.PHONY: require-secret
require-secret:
	@test -f $(PULL_SECRET) || { \
	  echo "No Red Hat pull secret at $(PULL_SECRET)"; \
	  echo ""; \
	  echo "  1. open https://console.redhat.com/openshift/create/local"; \
	  echo "  2. click 'Download pull secret'"; \
	  echo "  3. mv ~/Downloads/pull-secret.txt $(PULL_SECRET)"; \
	  echo ""; \
	  exit 1; }


##@ First-time setup

.PHONY: download
download: ## Fetch + checksum-verify the CRC installer, print the sudo line to run
	@cd $$(mktemp -d) && \
	  curl -LO $(MIRROR)/crc-macos-installer.pkg && \
	  curl -sL $(MIRROR)/sha256sum.txt | grep crc-macos-installer.pkg | shasum -a 256 -c - && \
	  echo "" && echo "now run:" && \
	  echo "  sudo installer -pkg $$PWD/crc-macos-installer.pkg -target /"

.PHONY: setup
setup: require-crc require-secret ## Configure CRC and boot the first cluster (~20 min from empty)
	crc config set preset openshift
	crc config set memory 16384
	crc config set cpus 6
	crc config set disk-size 60
	crc setup
	$(MAKE) -f $(THIS) start

##@ Suspend / resume  (cheap, keeps the VM and its state)

.PHONY: start
start: require-crc require-secret ## Resume the cluster.
	crc start -p $(PULL_SECRET)
	kubectl config use-context $(CTX)

.PHONY: stop
stop: require-crc ## Suspend the cluster (seconds). Frees RAM only.
	crc stop

.PHONY: status
status: require-crc ## CRC status and disk usage
	@crc status
	-@du -sh $(HOME)/.crc 2>/dev/null

##@ Destroy / rebuild  (~15 min to get back)

.PHONY: delete
delete: require-crc ## DESTROY the VM. Frees 30-45 GB. Keeps the cache so a rebuild needs no download.
	@crc status >/dev/null 2>&1 && crc delete -f || echo "no VM to delete"
	-@du -sh $(HOME)/.crc 2>/dev/null

.PHONY: recreate
recreate: delete ## DESTROY the VM and build a fresh cluster (~15 min, no download)
	$(MAKE) -f $(THIS) start

.PHONY: purge
purge: delete ## DESTROY the VM and the cache. Frees it all.
	rm -rf $(HOME)/.crc/cache
	-@du -sh $(HOME)/.crc 2>/dev/null

##@ Odigos

.PHONY: install
install: require-crc ## Install odigos on OpenShift. Enterprise when ODIGOS_TOKEN is set.
	helm --kube-context $(CTX) upgrade --install odigos $(CHART) \
	  -n $(ODIGOS_NS) --create-namespace \
	  --set openshift.enabled=true \
	  --set openshift.certifiedImageTags=false \
	  --set imagePrefix=$(IMAGE_PREFIX) \
	  --set image.tag=$(TAG) \
	  $(TOKEN_FLAG) \
	  $(if $(ODIGLET_IMAGE),--set images.$(ODIGLET_KEY)=$(ODIGLET_IMAGE)) \
	  $(HELM_ARGS)
	$(OC_BIN) --context $(CTX) rollout status ds/odiglet -n $(ODIGOS_NS) --timeout=300s

.PHONY: uninstall
uninstall: require-crc ## Remove odigos
	-helm --kube-context $(CTX) uninstall odigos -n $(ODIGOS_NS)

##@ Node access

.PHONY: node
node: require-crc ## Interactive root shell on the CRC node
	@test -f $(SSH_KEY) || { echo "No VM - run 'make -f $(THIS) start'"; exit 1; }
	@crc status >/dev/null 2>&1 || { echo "Cluster is not running - 'make -f $(THIS) start'"; exit 1; }
	@ssh -q -i $(SSH_KEY) -p $(SSH_PORT) -t \
	  -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
	  core@127.0.0.1 sudo -i

.PHONY: console
console: require-crc ## Print the web console URL and kubeadmin credentials
	crc console --credentials
