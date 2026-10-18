# Odigos Helm Chart

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![Release](https://github.com/odigos-io/odigos/actions/workflows/release.yml/badge.svg?branch=main)](https://github.com/odigos-io/odigos/actions/workflows/release.yml)

This repository contains the helm chart for Odigos - the observability control plane.

```
helm repo add odigos https://odigos-io.github.io/odigos --force-update
```

## Migration

If you have been previously using the `odigos-charts` chart repository you will get an error "Error: repository name (odigos) already exists, please specify a different name".
To update to new repository location run:

```sh
helm repo add odigos https://odigos-io.github.io/odigos/ --force-update
```

## Usage

Add helm repository:

```sh
helm repo add odigos https://odigos-io.github.io/odigos
```

### Install Odigos

```sh
helm repo update
helm upgrade --install odigos odigos/odigos --namespace odigos-system --create-namespace
```

- **Openshift Clusters** - Make sure to set `openshift.enabled=true` in the values file or pass it as a flag while installing the chart.
- **GKE Clusters** - Make sure to set `gke.enabled=true` in the values file or pass it as a flag while installing the chart.

#### Using a Custom Docker Registry

By default, community installs pull images from `docker.io/keyval` and enterprise installs (with an on-prem token) pull from `registry.odigos.io`. To use a custom Docker registry instead, set the `imagePrefix` value during installation:

```sh
helm upgrade --install odigos odigos/odigos \
  --namespace odigos-system \
  --create-namespace \
  --set imagePrefix=$CUSTOM_DOCKER_REGISTRY
```
Make sure to replace `$CUSTOM_DOCKER_REGISTRY` with the URL of your Docker registry.

For more details on configuring a custom Docker registry, refer to the [Docker Registry Setup Documentation](https://docs.odigos.io/setup/docker-registry).

### Upgrade an existing Interrogation proxy

Existing installations may have a Helm-managed `odigos-litellm` ConfigMap. Upgrade in two phases with compatible Instrumentor and Insights images so the new controller can take over that ConfigMap without interrupting the proxy:

```sh
export KUBE_CONTEXT=<your-kubernetes-context>
helm --kube-context "$KUBE_CONTEXT" upgrade odigos ./helm/odigos \
  --namespace odigos-system --reuse-values \
  --set insights.interrogation.litellm.bootstrapConfig=true
kubectl --context "$KUBE_CONTEXT" -n odigos-system get configmap odigos-interrogation-status \
  -o jsonpath='{.data.proxy}'
kubectl --context "$KUBE_CONTEXT" -n odigos-system get deployment odigos-litellm \
  -o jsonpath='{.spec.template.metadata.annotations.odigos\.io/ai-settings-revision}'
```

Wait for the proxy status to report `READY` and verify its revision matches the `odigos.io/ai-settings-revision` annotation on the LiteLLM Deployment. Then remove the temporary Helm template; the keep annotation prevents Helm from deleting the ConfigMap while the Instrumentor remains its owner:

```sh
helm --kube-context "$KUBE_CONTEXT" upgrade odigos ./helm/odigos \
  --namespace odigos-system --reuse-values \
  --set insights.interrogation.litellm.bootstrapConfig=false
```

If the first upgrade must be rolled back, use `helm --kube-context "$KUBE_CONTEXT" rollback odigos <previous-revision> --namespace odigos-system`. The previous chart resumes rendering the proxy ConfigMap and using the Helm-managed upstream credential. With Helm 4 server-side apply, append `--force-conflicts` to transfer the ConfigMap fields back from the controller; otherwise rollback reports a field ownership conflict. Keep compatible Instrumentor and Insights images when rolling back only the bootstrap/ownership transition.

### Uninstall Odigos

```sh
helm delete odigos -n odigos-system
kubectl delete ns odigos-system
```

## License

[Apache 2.0 License](https://github.com/prometheus-community/helm-charts/blob/main/LICENSE).
