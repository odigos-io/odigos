#!/bin/bash

set -e

if helm status chaos-mesh -n chaos-mesh >/dev/null 2>&1; then
  echo "chaos-mesh helm already installed, probably from previous run. Skipping..."
else
  helm repo add chaos-mesh https://charts.chaos-mesh.org
  helm repo update
  RUNTIME=$(kubectl get node "$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}')" -o=jsonpath='{.status.nodeInfo.containerRuntimeVersion}')
  if [[ $RUNTIME == docker://* ]]; then
      echo "Docker runtime detected."
      helm install chaos-mesh chaos-mesh/chaos-mesh -n chaos-testing --create-namespace
  elif [[ $RUNTIME == containerd://* ]]; then
      helm install chaos-mesh chaos-mesh/chaos-mesh -n=chaos-mesh --set chaosDaemon.runtime=containerd --set chaosDaemon.socketPath=/run/containerd/containerd.sock --create-namespace
  else
      echo "Error: Unsupported container runtime detected: $RUNTIME" >&2
      exit 1
  fi
fi
