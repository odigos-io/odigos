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

Once a new version of Odigos has been released, and the components have all passed OpenShift certification, do the following:

1. Update the `VERSION` variable at the top of the `Makefile`
2. Update the tag for each `RELATED_IMAGE_*` environment variable in `config/manager/manager.yaml`
3. In `config/manifests/bases/odigos-operator.clusterserviceversion.yaml`, update the following:
    1. The `version` in the `alm-examples` annotation
    2. The tag on the `containerImage` annotation
    3. The `metadata.name` version
    4. The `spec.version` value
4. Run `USE_IMAGE_DIGESTS=true make generate manifests bundle` to update the generated bundle and commit the results.

### Publishing the release to OpenShift

When you have verified the new version, open a pull request to https://github.com/redhat-openshift-ecosystem/certified-operators

Example: https://github.com/redhat-openshift-ecosystem/certified-operators/pull/5535

Do the following in your PR:

1. Create a new folder called `operators/odigos-operator/v<VERSION>`
2. Create 2 sub-folders in `operators/odigos-operator/v<VERSION>`:
    1. `manifests`
    2. `metadata`
3. Copy everything from `config/bundle/manifests` (in this repo) to your new `operators/odigos-operator/v<VERSION>/manifests` folder
4. Copy the `annotations.yaml` from a previous version of `operators/odigos-operator` to your new `metadata` folder
5. Open a pull request to the Red Hat repo with the title format: `operator odigos-operator (v<VERSION>)`
