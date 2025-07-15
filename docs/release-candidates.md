# Release Candidates

Odigos supports release candidates for Helm charts, allowing users to test pre-release versions without affecting the stable release channel.

## How It Works

When a release is tagged with a release candidate identifier (e.g., `v1.1.0-rc.1`), it gets published to a subdirectory in the main Helm repository that doesn't interfere with stable releases.

## Repository Structure

- **Stable releases**: `https://odigos-io.github.io/odigos/` (root of gh-pages)
- **Release candidates**: `https://odigos-io.github.io/odigos/rc/` (rc/ subdirectory)

## Release Candidate Detection

The release script automatically detects release candidates by looking for the `-rc` pattern in the tag:
- `v1.1.0-rc.1` ✅ (release candidate)
- `v1.1.0` ❌ (stable release)

## Usage

### For Users

To use release candidates:

```bash
# Add the release candidate repository
helm repo add odigos-rc https://odigos-io.github.io/odigos/rc/
helm repo update

# Install a specific release candidate
helm install odigos odigos-rc/odigos --version 1.1.0-rc.1
```

### For Developers

To create a release candidate (this get's run in CI automatically):

```bash
# Create and push a release candidate tag
git tag v1.1.0-rc.1
git push origin v1.1.0-rc.1

# Run the release script
TAG=v1.1.0-rc.1 GITHUB_REPOSITORY=odigos-io/odigos ./scripts/release-charts.sh
```

## Important Notes

⚠️ **Release candidates are pre-release versions and may contain bugs or incomplete features.**
⚠️ **Do not use release candidates in production environments.**

Release candidates are intended for:
- Testing new features
- Validating bug fixes
- Getting early feedback from the community
- CI/CD pipeline testing

## Migration Path

When a release candidate is ready for production:

1. Create a stable release tag (e.g., `v1.1.0`)
2. Release automation runs and releases the stable version
3. The stable version will be available in the main repository

Users can then upgrade from the RC to the stable version:

```bash
# Remove RC repository
helm repo remove odigos-rc

# Add stable repository (if not already added)
helm repo add odigos https://odigos-io.github.io/odigos/
helm repo update

# Upgrade to stable version
helm upgrade odigos odigos/odigos --version 1.1.0
```

## Repository Structure Details

The `gh-pages` branch will contain:

```
gh-pages/
├── index.yaml          # Stable releases index
└── rc/
    ├── index.yaml      # Release candidates index
    ├── README.md       # RC usage instructions
```

This approach keeps everything in one repository while providing clear separation between stable and pre-release versions. 