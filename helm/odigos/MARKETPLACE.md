# AWS Marketplace Enterprise contracts

The Marketplace activation profile selects Enterprise components using the buyer's AWS contract entitlement. It requires the matching Enterprise release with AWS License Manager support. Existing community and Odigos token installations keep their current behavior.

The Marketplace release must provide a values file containing all of these explicit image references:

- `images.autoscaler`
- `images.scheduler`
- `images.enterprise-instrumentor`
- `images.enterprise-odiglet`
- `images.enterprise-collector`
- `images.enterprise-ui`
- `images.enterprise-agents` (also injected into customer workloads)
- `images.cli` (Helm uninstall cleanup)

Use the image URIs and tags published with the selected Marketplace version. They must refer to its Marketplace-managed ECR repositories. An omitted component fails rendering instead of falling back to a community or private-registry image.

The deployment values additionally contain:

```yaml
marketplace:
  enabled: true
  licenseManagerRegion: us-east-1
  serviceAccountAnnotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::BUYER_ACCOUNT:role/ODIGOS_LICENSE_ROLE
```

The region must match the buyer's License Manager entitlement. This example uses IRSA. Configure the role trust policy for the selected EKS cluster's OIDC provider and these service accounts in the installation namespace:

- `odigos-instrumentor`
- `odiglet`
- `odigos-ui`
- `odigos-gateway`

Grant `license-manager:CheckoutLicense` and `license-manager:CheckInLicense`. The paired Enterprise implementation checks a non-counted `enterprise` tier using a product SKU bound into the binaries at build time. Do not supply access keys or an Odigos on-prem token. Supported EKS Pod Identity associations can be configured separately instead of using IRSA annotations.

The autoscaler propagates the Marketplace provider and region to gateway collector pods it creates. It does not itself need License Manager IAM permissions. The `odigos-pro` Secret contains only an edition marker; it is not proof of entitlement and cannot activate the Enterprise binaries without a successful AWS check.

The initial profile covers core Enterprise instrumentation and exporting telemetry. Insights is explicitly rejected until its Marketplace licensing and lifecycle are supported. Do not combine Marketplace activation with Odigos token or registry-secret activation.

Validation:

```sh
helm lint helm/odigos
python3 tests/e2e/helm-chart/check-marketplace.py
cd autoscaler
go test ./controllers/clustercollector -run '^TestCollectorLicenseEnvironment$'
```

The render test needs PyYAML and Helm and uses synthetic image names; it never contacts AWS or installs workloads. Before publication, test the actual images, buyer entitlement, IAM roles, dynamic gateway collectors, upgrade and uninstall in EKS.

This profile supports **Helm delivery**. EKS managed add-on delivery additionally needs a package without unsupported Helm hooks/lookups, a supported configuration schema and AWS add-on certification. Do not submit the ordinary chart as an add-on unchanged.

See [AWS contract integration](https://docs.aws.amazon.com/marketplace/latest/userguide/container-license-manager-integration.html) and [EKS add-on requirements](https://docs.aws.amazon.com/marketplace/latest/userguide/container-product-policies.html).
