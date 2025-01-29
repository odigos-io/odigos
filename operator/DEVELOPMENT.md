# Development and testing

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
make generate manifests bundle
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

### Running in OpenShift

To test in an OpenShift cluster, push the operator image and bundle image to your registry.

Connect to the cluster with `oc login` and run:

```
operator-sdk run bundle <path to bundle image:tag>
```

Clean up with:

```
operator-sdk cleanup odigos-operator  --delete-all
```
