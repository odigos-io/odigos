# EKS managed add-on package

Build a separate core Enterprise add-on from the ordinary chart. The package reuses instrumentor-only IRSA/MeterUsage and the purchase-verified license issuer. Ordinary Helm templates and non-Marketplace runtime behavior stay unchanged. This is a candidate until AWS ingestion and live add-on certification complete.

## Build

Requires Helm >=3.19, Python 3, PyYAML 6.0.3 and jsonschema 4.25.1 for tests. Supply exactly eight image references in Marketplace ECR: `autoscaler`, `scheduler`, `instrumentor`, `odiglet`, `collector`, `ui`, `agents`, `cli`.

```sh
python3 helm/eks-addon/build.py --version "$RELEASE_VERSION" \
  --images "$IMAGE_MANIFEST" --namespace odigos-system \
  --kube-version "$EKS_KUBERNETES_VERSION" --output "$NEW_OUTPUT_DIRECTORY"
```

The builder emits a `chart` directory, Helm archive, image/release manifest, offline readiness report and cleanup job. It requires a stable semantic version and a fresh output directory. It never pushes images, changes listings or deploys infrastructure. Build Enterprise images with the existing Marketplace public key; bind the product code only into instrumentor. Build scheduler from this branch to include state initialization. Verify every image on AMD64 and ARM64 before public release.

The generated package removes unsupported features and lookups. Named image repository fields support AWS regionalization. `aws_mp_configuration_schema.json` permits log level, resource sizing and node selectors. EKS supplies cluster name through the release manifest's `EnvironmentOverrideParameters`. Secrets remain outside add-on configuration. The chart creates `odigos-instrumentor`; EKS binds its IRSA role using `serviceAccountRoleArn`. No Pod Identity metadata file is emitted because AWS's current format supports Pod Identity only and PAYG requires IRSA. Confirm role injection during Limited ingestion.

## State and lifecycle

Only the EKS package sets `ODIGOS_INSTALLATION_METHOD=eks-addon`. Scheduler initializes missing deployment-ID/Go-offset keys once with optimistic concurrency, preserving existing values. Odiglet waits for the offsets key. This adds no background loop or AWS dependencies to odiglet. Validate EKS managedFields, repeated reconciliation, upgrades/rollback and custom offsets after ingestion; unit tests alone do not establish EKS field ownership.

[the shared installation guide](../../docs/snippets/shared/setup/eks-addon.mdx) (packaged as `INSTALL.md`) contains purchasing, token retrieval/renewal, install/update and cleanup-before-delete instructions. Cleanup is explicitly run while controllers are active. The opt-in cleanup template is omitted from all normal add-on renders. Direct console/API deletion skips it; acceptance of that documented lifecycle remains part of AWS review. Do not implement cleanup in a pod shutdown hook that depends on concurrently deleted controllers/RBAC.

## Verification and release

```sh
python3 scripts/test-eks-addon-readiness.py
python3 helm/eks-addon/test_build.py
python3 tests/e2e/helm-chart/check-marketplace.py
(cd scheduler && go test -race ./clusterinfo ./controllers/odigospro)
```

Keep release artifacts separate from the public Helm version. Submit a new Limited EKS delivery option on the same paid product. After ingestion, test actual add-on APIs in the seller account in Seoul, every claimed Kubernetes version and both architectures. Verify telemetry, metering, renewal, IAM denial, state preservation and safe removal. Provide AWS's required internal test offers and retain the test environment until public approval. Host the installation guide on the vendor documentation site before public release.

[AWS requirements](https://docs.aws.amazon.com/marketplace/latest/userguide/container-product-policies.html) · [submission](https://docs.aws.amazon.com/marketplace/latest/userguide/container-add-version.html) · [certification](https://docs.aws.amazon.com/marketplace/latest/userguide/test-release-product.html)
