# Versioning

Odigos publishes 2 release types:
-  stable releases (`v1.X.Y`)
-  release candidates (`v1.X.0-rcY`)
-  pre-release versions (`v1.X.0-preY`)

Stable releases are the default and recommended version to install. Release candidates and pre-release versions are opt-in and require the user to explicitly take an action to install them.

## Consuming Odigos Versions

Odigos should always default to using a stable version for any installation method. The instllation options are mentioned in [the docs](https://docs.odigos.io/setup/installation).

To install a release candidate version, one needs to "hop-in" to the release candidate version.

- **brew** - use the "rc" tap (`homebrew-odigos-cli-rc/odigos`) which is not the default tap. this will always install the latest release candidate version. **Notice**, the tap for stable releases is `homebrew-odigos-cli/odigos`, so pay attention to the tap name and the version you are using to avoid installing the wrong version.

```bash
brew update && brew install odigos-io/homebrew-odigos-cli-rc/odigos
```

- **helm repo** - release candidates are published to the same repo and are ignored by helm by default. to install a release candidate, first `helm repo update` then use `--version` flag to specify the release candidate version.

```bash
helm repo add odigos https://odigos-io.github.io/odigos
helm repo update
helm install odigos odigos/odigos --version v1.0.X-rcY
```

- **cli binray** - browse to https://github.com/odigos-io/odigos/releases and download the cli binary for the specific release candidate version.

```bash
curl -L https://github.com/odigos-io/odigos/releases/download/v1.0.X-rcY/cli_1.0.X-rcY_linux_amd64.tar.gz -o odigos.tar.gz
tar -xzf odigos.tar.gz
chmod +x odigos
./odigos version
```

- **helm chart** - download the latest release candidate version from the [github releases](https://github.com/odigos-io/odigos/releases) page and install it with `helm install odigos odigos-1.0.X-rcY.tgz`

```bash
curl -L https://github.com/odigos-io/odigos/releases/download/v1.0.X-rcY/helm-chart-odigos-1.0.X-rcY.tgz -o odigos-1.0.X-rcY.tgz
helm install odigos odigos-1.0.X-rcY.tgz
```

- **cli image** - pull the cli image from the registry with the specific "-rcY" version tag.

```bash
docker run --rm -it registry.odigos.io/odigos-cli:v1.0.X-rcY version
```
