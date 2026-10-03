# AWS Marketplace Helm delivery

This chart keeps the existing Odigos Enterprise license token and normal licensing UI. It adds image checks, instrumentor's IRSA identity and namespaced ConfigMap permissions for centralized Marketplace billing. Odiglet needs no Marketplace identity or additional permissions. It is not a managed EKS add-on package.

Obtain a Marketplace token through the seller's authenticated token service after subscribing. Use AWS credentials from the purchasing account with `lambda:InvokeFunctionUrl` and `lambda:InvokeFunction` permission on the published function ARN. The service verifies the purchase and signs a Marketplace-only license; a normal Enterprise token does not unlock Marketplace images. The EKS deployment account must also have an active Marketplace entitlement; a token obtained by another account does not grant metering access.

The final release instructions must supply the deployed token URL and function ARN. With Python and boto3 installed:

```sh
python3 helm/odigos/fetch-marketplace-token.py "$ODIGOS_TOKEN_URL" --profile buyer --output odigos-marketplace-token
kubectl create namespace odigos --dry-run=client -o yaml | kubectl apply -f -
kubectl -n odigos create secret generic odigos-pro --from-file=odigos-onprem-token=odigos-marketplace-token --dry-run=client -o yaml | kubectl apply --server-side -f -
```

The helper writes a mode-0600 file and does not print the token. Keep it out of source control and shell history. The token lasts up to 30 days, capped at the agreement end. Before expiry, retrieve a new token into a new file, update the Secret, and roll the token-consuming Odigos workloads (including collector sidecars); an updated Secret alone does not update environment variables in existing pods. For gateway collectors managed by the autoscaler, recreate pods one at a time and wait for readiness; the autoscaler reconciles away Deployment restart annotations. Renewal repeats purchase verification. A renewal controller is not included.

Install with the existing `odigos-pro` Secret and `externalOnpremTokenSecret: true`:

```yaml
externalOnpremTokenSecret: true
marketplace:
  enabled: true
  serviceAccountAnnotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::BUYER_ACCOUNT:role/ODIGOS_MARKETPLACE_ROLE
```

Set all eight `images` entries to the release's Marketplace ECR URIs: `autoscaler`, `scheduler`, `enterprise-instrumentor`, `enterprise-odiglet`, `enterprise-collector`, `enterprise-ui`, `enterprise-agents` and `cli`. For Marketplace packaging, use a `repository` object for each override, with the complete tagged image URL, for example `images.autoscaler.repository: 709825985650.dkr.ecr.us-east-1.amazonaws.com/odigos/autoscaler:RELEASE`. AWS uses these named repository fields to validate and regionalize images; existing plain-string overrides remain supported outside that packaging. Images must be available from AWS Marketplace, including injected agents and uninstall cleanup. Build instrumentor with the Marketplace product code and signing public key. Odiglet, collector and UI must also be built with that public key; the other images can be mirrored from the same compatible validated release. Never bind the Marketplace key into the private registry or ordinary Enterprise images. Do not mix untested release versions.

For AWS-managed Helm launch, configure the paid-product override `marketplace.serviceAccountName` with default `${AWSMP_SERVICE_ACCOUNT}` in the Add Version delivery metadata. When this value is provided, the chart uses the existing service account for instrumentor and its role bindings and does not create or modify it. That account must already have IRSA configured in the installation namespace. The manual installation above instead creates `odigos-instrumentor` with the supplied role annotation.

Configure IRSA trust only for the selected instrumentor service account (by default `odigos-instrumentor`) in the installation namespace, with the EKS OIDC provider and `sts.amazonaws.com` audience. Grant `aws-marketplace:MeterUsage` on `Resource: "*"`. IRSA supplies the region and web identity token. Node roles, static access keys and EKS Pod Identity are unsupported by this API.

The product-bound instrumentor image always meters; the chart flag configures installation, not a billing off switch. It checks the subscription at startup. Both replicas observe Ready, Running odiglet containers owned by the installation's DaemonSet, counting each distinct node once per UTC hour, including observed partial hours. One elected reporter sends the previous closed hour as an aggregate `host` quantity. A durable ConfigMap claim freezes each charge before the AWS request. A replacement pod skips uncertain charges to avoid duplicate billing; outages can therefore undercollect a cluster-hour. Billing never runs in odiglet.

Preserve `odigos-mp-*` ConfigMaps across same-hour reinstall, including when retaining the installation namespace. They are intentionally excluded from workload garbage collection and the standard Odigos uninstall selector; deleting the namespace also deletes this protection. The running reporter prunes expired records. After final removal, leftover records can be deleted after two full UTC hours. Removal before the next hourly report can leave the final hour unbilled. Only one Odigos installation per cluster is supported by this delivery.

Run `helm lint helm/odigos` and `python3 tests/e2e/helm-chart/check-marketplace.py` (Helm and PyYAML required). These render checks cover existing token/community modes, explicit images, only instrumentor receiving IRSA, its pod identity and namespaced ledger permissions, and unchanged odiglet permissions. Real token issuance and renewal, AWS subscription, billing, scans and EKS lifecycle tests are still required before publication. Insights is outside the initial delivery.

[Container requirements](https://docs.aws.amazon.com/marketplace/latest/userguide/container-product-policies.html) · [Metering requirements](https://docs.aws.amazon.com/marketplace/latest/userguide/container-metering-meterusage.html)
