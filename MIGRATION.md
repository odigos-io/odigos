# Migration

Odigos is composed of two parts:
1. The Odigos cli and UI.
1. The Odigos deployment in a k8s cluster.

## Check your Versions

Use Odigos cli to test for your versions of odigos:
```bash
➜  ~ odigos version
Odigos Cli Version: version.Info{Version:'v0.1.81', GitCommit:'c79f5fa', BuildDate:'2023-10-19T06:59:08Z'}
Odigos Version (in cluster): version.Info{Version:'v0.1.81'}
```

This cli output (at the time of writing) shows that the cli version is `v0.1.81` and the version of Odigos in the cluster is also `v0.1.81`. It is recommended that these versions match.

## Upgrade Odigos CLI

### Brew (MacOS only)

```sh
brew install keyval-dev/homebrew-odigos-cli/odigos
```

Will install the latest version of the Odigos cli from brew.

### GitHub Releases

Go to the [Releases Page](https://github.com/keyval-dev/odigos/releases), download the latest version for your arch and os, and install it. Make sure it's in your `PATH`. You can execute `odigos version` in your shell to verify that it's installed correctly.

## Upgrade Odigos in the Cluster

** New from Odigos v0.1.81 **

```sh
$ odigos migrate
```

to upgrade (or downgrade) to a specific version, use the `--version` flag:

```sh
$ odigos migrate --version v0.1.81
```
