# Development and testing

Running the operator requires the following steps:

1. Build the Operator image (the controller)
2. Generate the Operator bundle (a collection of ready-to-deploy manifests)
3. Build the Operator Bundle Image (another image used by OpenShift to deploy the manifests)

The current version of Odigos is set by the `VERSION` variable in the Makefile. This value
reflects what version of component images will be installed in the cluster. This should stay
bound to the version of the operator.

## Building the Operator image

From the root repo directory (ie `cd ..` from here), run:

```
make build-operator
```

You can build a test image by setting `ORG` and `TAG` build arguments, eg:

```
make build-operator ORG=mikeodigos TAG=dev
```

If you are on Mac and want to test this in an OpenShift cluster running on Linux, use the
`make push-operator` command instead (which uses `docker buildx` to cross-compile).

## Building the Operator Bundle

For OpenShift, we need an Operator Bundle which is an image consisting of the manifests
and metadata required to install the Operator.

From this directory, run:

```
USE_IMAGE_DIGESTS=true make generate manifests bundle
```

You can set your dev image by passing the `IMG` build arg:

```
make bundle IMG=docker.io/mikeodigos/odigos-operator:dev
```

Note that the fully-qualified domain is required (`docker.io`). This will rewrite the manifests
with Kustomize to point to your docker image.

Next, build the Bundle Image:

```
make bundle-build
```

Again, you can point to your image by setting `IMAGE_TAG_BASE`:

```
make bundle-build IMAGE_TAG_BASE=mikeodigos/odigos-operator VERSION=dev
```

This will build a new docker image for the bundle containing all the generated manifests.

### Building with custom component images

If you are testing custom components (eg, Odiglet, Instrumentor, etc), do the following:

1. Build and push your custom images to your Docker registry
2. Update each `RELATED_IMAGE_*` environment variable in `config/manager/manager.yaml`
3. Run `make generate manifests bundle`
4. Run `make bundle-build IMAGE_TAG_BASE=<your-registry>/odigos-operator VERSION=dev` to build a new bundle image
5. Push the bundle image to your registry and run it with `operator-sdk run bundle` (see below)

### Running in OpenShift

To test in an OpenShift cluster, push the operator image and bundle image to your registry.

Connect to the cluster with `oc login` (copy login command from OpenShift console) and run:

```
operator-sdk run bundle <path to bundle image:tag> -n odigos-operator-system
```

Clean up with:

```
operator-sdk cleanup odigos-operator  --delete-all
```

### ImagePull error on OpenShift

To authenticate with your DockerHub account on OpenShift, follow [this OpenShift support page](https://access.redhat.com/solutions/6159832).

## Preparing a new release

### Releasing RHEL images

For an existing Odigos release, run the [Publish Modules for RHEL action](https://github.com/odigos-io/odigos/actions/workflows/publish-modules-rhel.yml)
with the image tag you want to release on RHEL (eg, `v1.24.2`). It can be run from `main`, since it uses the provided tag for build.

If this completes successfully, it should trigger the [Release Enterprise Components to Artifact Registry (RHEL) action](https://github.com/odigos-io/odigos-enterprise/actions/workflows/release-images-rhel.yml)
in odigos-enterprise.

When the RHEL release finishes, there should be images for all components with a `-rhel-certified` suffix in [Artifact Registry](https://console.cloud.google.com/artifacts/docker/odigos-cloud/us-central1/components?project=odigos-cloud).

### OpenShift Certification

Verify that all components pass OpenShift certification by running the [OpenShift certification for container images action](https://github.com/odigos-io/odigos/actions/workflows/openshift-preflight.yml)
with the dry-run flag checked (`Run preflight checks only; do not submit results to Red Hat`). If that succeeds, run the job again without dry run
to actually submit to Red Hat.

If it fails for any images, fix the failures and manually re-push the affected images to Artifact Registry using the provided
Makefile `push-image` targets with `RHEL` and `PUSH_IMAGE` flags.

You can then re-run certification for a specific component using the check boxes in the OpenShift certification action.

### Updating manifests and bundle

Once a new version of Odigos has been released, and the components have all passed OpenShift certification, do the following:

1. Update the `VERSION` variable at the top of the `operator/Makefile`
2. Update the tag for each `RELATED_IMAGE_*` environment variable in `operator/config/manager/manager.yaml`
3. In `operator/config/manifests/bases/odigos-operator.clusterserviceversion.yaml`, update the following:
    1. The `version` in the `alm-examples` annotation
    2. The tag on the `containerImage` annotation
    3. The `metadata.name` version
    4. The `spec.version` value
4. Run `USE_IMAGE_DIGESTS=true make generate manifests bundle` from the `operator/` directory to update the generated bundle and commit the results.

### Publishing the release to OpenShift

When you have verified the new version, open a pull request to https://github.com/redhat-openshift-ecosystem/certified-operators

Example: https://github.com/redhat-openshift-ecosystem/certified-operators/pull/5535

Do the following in your PR:

1. Create a new folder called `operators/odigos-operator/v<VERSION>`
2. Create 2 sub-folders in `operators/odigos-operator/v<VERSION>`:
    1. `manifests`
    2. `metadata`
3. Copy everything from `bundle/manifests` (in this repo) to your new `operators/odigos-operator/v<VERSION>/manifests` folder
4. Copy the `annotations.yaml` from a previous version of `operators/odigos-operator` to your new `metadata` folder
5. Open a pull request to the Red Hat repo with the title format: `operator odigos-operator (v<VERSION>)`
