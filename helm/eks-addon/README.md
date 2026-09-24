# EKS managed add-on submission candidate

**Later delivery option:** the initial launch uses the existing node-metered product through Helm. This candidate currently targets the alternative License Manager contract model and Pod Identity. It must be adapted to IRSA/MeterUsage and certified before use with the selected node product; it is not a gate on the first Helm private offer.

This builds a separate core Enterprise chart from `helm/odigos`. Ordinary Helm installation behavior is unchanged. The output is an **offline candidate**, not a certified add-on or evidence that its images exist. Use a release containing the paired Marketplace licensing, Helm/autoscaler and scheduler changes.

## Build

Requirements: Helm 3.19 or later, Python 3, PyYAML 6.0.3 with LibYAML, and jsonschema 4.25.1 for tests.

Create an image JSON file with exactly these keys: `autoscaler`, `scheduler`, `enterprise-instrumentor`, `enterprise-odiglet`, `enterprise-collector`, `enterprise-ui`, `enterprise-agents`, `cli`. Each value must be the selected release's explicit tag or digest in AWS Marketplace's managed ECR registry. No synthetic image is supplied for installation. Confirm all eight artifacts, both architectures and scan results before submission.

```sh
python3 helm/eks-addon/build.py \
  --version "$RELEASE_VERSION" \
  --images "$IMAGE_MANIFEST" \
  --namespace odigos-system \
  --kube-version "$EKS_KUBERNETES_VERSION" \
  --output "$NEW_OUTPUT_DIRECTORY"
```

The version is a stable release number without `v`; selecting a number does not select or build the corresponding source. Run from the reviewed release checkout. The output directory must not exist. The build emits the chart, `.tgz`, full image manifest, offline preflight and a separate cleanup job. It does not contact AWS, push images or deploy a cluster. The Kubernetes argument is a rendering target, not a certification claim.

The package removes Insights, Central and GKE templates, replaces live lookups, excludes the Helm uninstall hook and includes the AWS configuration and Pod Identity files. It retains all normal core resources, CRDs and cleanup RBAC. A static scan, repeat rendering, image inventory and Helm lint must pass before packaging succeeds. The preflight is partial; runtime-created resources still need inspection in EKS.

When assembling `EksAddOnDeliveryOptionDetails` for the Catalog API submission, copy `environmentOverrideParameters` from `release-manifest.json` into `EnvironmentOverrideParameters`. EKS injects `clusterName` using `${AWS_EKS_CLUSTER_NAME}`; cluster metadata must not be a buyer-entered configuration property. The entitlement region is separate from the cluster region and must not be replaced with `${AWS_REGION}` for contract licensing.

## Runtime state

The add-on declares the deployment and Go-offset ConfigMaps without owning the mutable installation ID or offsets key. The scheduler initializes only missing keys with optimistic concurrency. A new installation uses the deployment ConfigMap's Kubernetes UID as its identity; an existing ID or custom offsets are retained. The odiglet volume requires the offsets key so pods wait for initialization instead of starting with a missing file. No additional RBAC permissions are introduced.

AWS uses server-side apply for add-on resources. Inspect `managedFields` after ingestion: `eks` must not own the installation ID or offsets data. Test upgrades with nonempty custom offsets and an existing ID, and verify the values survive both normal and conflict-resolving add-on updates. Unit tests prove initialization behavior, not AWS field ownership. See [EKS field management](https://docs.aws.amazon.com/eks/latest/userguide/kubernetes-field-management.html).

Initial scope is a **new installation** in a dedicated namespace on Linux EKS nodes. Do not use overwrite to take over an existing Helm/CLI installation without a tested migration. Insights, Central, automatic Go-offset updater images, own-telemetry storage and arbitrary image overrides are outside this candidate's configuration schema. Custom offsets can be updated manually through the matching CLI. The small initial schema exposes log level and licensing region; expand supported scheduling/resource settings only with validation.

## Licensing and IAM

Bind the confirmed contract product SKU into the Enterprise binaries. Set `marketplace.licenseManagerRegion` to the region containing the buyer's entitlement. Create Pod Identity associations for `odigos-instrumentor`, `odiglet`, `odigos-ui` and `odigos-gateway` in the chosen namespace, with the EKS Pod Identity agent installed. The gateway account covers collector pods created later by the autoscaler. Test every identity, including dynamically created gateway pods.

The submitted parameters reference AWS's [License Manager consumption policy](https://docs.aws.amazon.com/aws-managed-policy/latest/reference/AWSLicenseManagerConsumptionPolicy.html), which also includes GetLicense and ExtendLicenseConsumption. The implementation uses CheckoutLicense and CheckInLicense only; buyers may supply a narrower role permitting those actions. Complete real subscription and identity tests before claiming this delivery method works. No access keys or Odigos token are accepted through add-on configuration.

## Required cleanup before deletion

EKS does not run Helm pre-delete hooks. The release therefore ships `cleanup-job.yaml` **outside** the chart. While the add-on, subscription and instrumentor are still active, apply this job from the exact installed release:

```sh
kubectl apply -f "$RELEASE_DIRECTORY/cleanup-job.yaml"
kubectl wait -n odigos-system --for=condition=complete job/cleanup-job --timeout=300s
kubectl logs -n odigos-system job/cleanup-job
```

Use the namespace recorded in `release-manifest.json` if different. Check job completion and errors before proceeding. If automatic application rollout is disabled, manually roll out previously instrumented workloads and confirm the replacement pods have no Odigos instrumentation before deleting the add-on. A completed cleanup job alone does not prove this in that configuration. Retry failed cleanup by inspecting the failure and recreating only the cleanup job; keep the add-on running until cleanup succeeds.

Only then call `aws eks delete-addon` for the installed add-on. Direct deletion through the console/API skips this procedure. AWS and the release owner must review this lifecycle requirement; if one-click deletion is required, implement and test a controller-supported cleanup lifecycle before publication. Do not replace this requirement with a deletion-time pod hook, which cannot guarantee that the instrumentor and RBAC still exist.

## Release gates

- Pass tests: `python3 scripts/test-eks-addon-readiness.py`, `python3 helm/eks-addon/test_build.py`, and `cd scheduler && go test -race ./clusterinfo ./controllers/odigospro`.
- Complete seller verification, contract product setup and Marketplace license integration; test the buyer entitlement and IAM identities.
- Publish and declare every image/chart artifact; test AMD64 and ARM64, vulnerability clearance, expected cluster scale and licensing quotas.
- Test EKS install, telemetry, dynamic collectors, custom offsets, repeated reconciliation, upgrade/rollback and cleanup/delete. Confirm the supported Kubernetes/node matrix.
- Include the Marketplace UI fix from public PR #5872 in the paired release.
- Obtain AWS Limited ingestion, complete the AWS-requested tests, then request public visibility. Neither this package nor a passing local preflight performs that process.

AWS references: [add-on requirements](https://docs.aws.amazon.com/marketplace/latest/userguide/container-product-policies.html), [version submission](https://docs.aws.amazon.com/marketplace/latest/userguide/container-add-version.html), [testing and public release](https://docs.aws.amazon.com/marketplace/latest/userguide/test-release-product.html).
