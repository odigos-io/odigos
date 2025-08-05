# Versioning

## Release Types

### Stable Releases

Version of the form `v1.0.X` where X is a number, is a stable release.

This is the default and recommended version to install. For production clusters, it is recommended to use only stable releases.

These are versions which run the full test suite, and also been manually verified for few days by odigos team and the community.

Stable releases are published once a week on Sunday, if no blockers are found.

### Release Candidates

Version of the form `v1.0.X-rcY` where X is a number and Y is a number starting from 0, is a release candidate.

Release candidate for the next stable release is published once a week on Thursday. It includes the current state of odigos "main" branch at the time of the release candidate first initiation.

Once a release candidate is published, it is used, tested, and verified by odigos team and the community. Any issue found during this process should be fixed and cherry-picked to the release branch, and a new release candidate should be published that includes the fix.

Any new feature, or non-critical bug fixes during the release candidate period are merged to "main" branch normally and scheduled for the following stable release.

### "Preview" or "Pre-release" Versions

Version of the form `v1.0.X-prY` where X is a number and Y is a number starting from 0, is a "preview" or "pre-release" version.

These versions allow a user to opt-in and test the latest features and bug fixes which are not yet scheduled for any stable release.

Preview versions are published directly off the "main" branch, and are thus less stable. They should be used for testing only when the cutting edge features and bug fixes are needed.

Preview versions are published on demand by odigos team based on the need.


## Release Process

### Stable Release Process

The “next stable” version is currently defined as the latest version with its patch number increased by one.
(In the future, this definition will change to follow proper semantic versioning.)

A stable version is a release candidate that has successfully completed the release‑candidate process and is then promoted to stable.

To promote a release candidate to a stable release, trigger the [promote-to-stable](https://github.com/odigos-io/odigos/actions/workflows/promote-to-stable.yml) workflow with the tag of the release candidate.

Promoting a release candidate to a stable release will:

- Copy all existing images of the relevant release-candidate tag to the new stable tag.
- Create a new stable tag (on the same commit as the release candidate tag)
- Release odigos-cli (with the relevant embedded version and date)
- Create a new release on github with changelog based on latest stable.
- Trigger OpenShift Preflight and helm release.

### Release Candidate Process

Use the [create-release-candidate](https://github.com/odigos-io/odigos/actions/workflows/create-release-candidate.yml) workflow to create a new release candidate for the next stable version.

Release candidates are made from release branch in the formath "releases/v1.0.X". the first time a release candidate is triggered for a stable version, the release branch is created off the "main" branch and any future commits to the "main" branch will not be included by default in the next release.

The workflow will check what the next stable version is, and if a release branch already exists for that version, it will use it. It will calculate the next release candidate number, and create a "Release Candidate Pull Request" to the release branch.

This Pull Request will run the full suite of tests and validations, and requires an approver to review and approve it. Once merged, the CI will create and push a tag for the release candidate, which will trigger odigos release process.

### Preview or Pre-release Process

Currently it is triggered manually by pushing a tag to the repo.

When triggering this release, check what the next stable version is (not the current stable), and check if there are any existing "-pr" tags. then use "-pr0" for first preview, or increment the number for the next preview.

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
