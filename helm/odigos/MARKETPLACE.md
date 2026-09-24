# AWS Marketplace Enterprise activation

The initial delivery uses Helm on EKS and the existing node-metered product. Private offers can provide negotiated annual commitments and excess-use prices. Enterprise images validate the AWS subscription. Community and existing token installations retain their behavior.

## Node-metered deployment

Use the image URIs and tags published with the Marketplace release. All eight entries must explicitly reference its Marketplace-managed ECR images; missing any fails rendering:

- `images.autoscaler`
- `images.scheduler`
- `images.enterprise-instrumentor`
- `images.enterprise-odiglet`
- `images.enterprise-collector`
- `images.enterprise-ui`
- `images.enterprise-agents` (also injected into customer workloads)
- `images.cli` (Helm uninstall cleanup)

```yaml
marketplace:
  enabled: true
  billingModel: node
  serviceAccountAnnotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::BUYER_ACCOUNT:role/ODIGOS_MARKETPLACE_ROLE
```

Configure OIDC role trust for the EKS cluster, installation namespace, `sts.amazonaws.com` audience and these service accounts:

- `odigos-instrumentor`
- `odiglet`
- `odigos-ui`
- `odigos-gateway`

Grant `aws-marketplace:MeterUsage` on `Resource: "*"`. IRSA supplies regional AWS environment and a projected identity token. **MeterUsage requires IRSA for EKS; Pod Identity, node roles and static keys are unsupported.** Node mode requires the role annotation; `licenseManagerRegion` is unused.

The paired Enterprise images must bind the selected product code. Each process performs a startup subscription dry-run; only odiglet reports positive host usage. Its pod UID comes from the Downward API. A Node billing checkpoint requires Node update permission even with `k8s-init-container` mounting.

The proposed billing basis is one node with an active Odigos agent per UTC hour, partial hours rounded up, irrespective of instrumented application count. Publish this definition in usage terms. Pod replacements use the checkpoint to avoid duplicate charges; uncertain hours are skipped and may be undercollected. Preserve `marketplace.odigos.io/host-*` annotations across reinstall in the same hour. Normal CLI cleanup leaves them intact. Real subscription, billing and lifecycle tests are required before release.

The autoscaler forwards the provider and billing model to gateway collectors; it needs no Marketplace IAM permission itself. The `odigos-pro` Secret is only an edition marker and cannot authorize Enterprise binaries without AWS validation. Mixed token/registry-secret activation is rejected. Initial scope is core instrumentation/export; Insights is rejected until supported separately.

## Alternative contract product

For a separate License Manager contract product, use `billingModel: contract` (the default), set `licenseManagerRegion` to the entitlement region and grant `license-manager:CheckoutLicense` and `license-manager:CheckInLicense`. Matching images must bind that product's SKU and no metering code. This checks a non-counted `enterprise` entitlement. IRSA or supported Pod Identity can be used for this contract path. Do not select it for the existing custom-metered product.

## Verification

```sh
helm lint helm/odigos
python3 tests/e2e/helm-chart/check-marketplace.py
go -C autoscaler test ./controllers/clustercollector -run '^TestCollectorLicenseEnvironment$'
```

The render test needs PyYAML and Helm and uses synthetic images. It checks both billing modes, identities, collector configuration, pod UID and Node permissions without AWS calls. Before publication, test actual images, buyer subscription, denied IAM, collector scaling, billing quantities and install/upgrade/uninstall on EKS.

Managed EKS add-on delivery needs a separate package without unsupported Helm hooks/lookups, a supported schema and AWS certification. The ordinary chart cannot be submitted unchanged.

References: [MeterUsage](https://docs.aws.amazon.com/marketplace/latest/userguide/container-metering-meterusage.html), [private offers](https://docs.aws.amazon.com/marketplace/latest/userguide/private-offers-supported-product-types.html), [contract integration](https://docs.aws.amazon.com/marketplace/latest/userguide/container-license-manager-integration.html), [EKS requirements](https://docs.aws.amazon.com/marketplace/latest/userguide/container-product-policies.html).
