# Odigos Open Source documentation

This directory powers **https://docs.odigos.io** on Mintlify and covers **Odigos Open Source only**. [Odigos Enterprise documentation](https://enterprise.docs.odigos.io) lives in `odigos-enterprise/docs` and requires a valid, unexpired generated enterprise API key.

## Editing and previewing

```sh
cd docs
node scripts/check-docs.mjs
npx mint@4.2.876 dev
```

Keep the permanent Open Source banner, Enterprise navbar/sidebar links, and edition notices on overview, quickstart, installation, and instrumentation pages. Enterprise-only features may have short public explanations linking to the protected guide; implementation and setup instructions belong in the enterprise repo.

`enterprise/`, `central/`, `vmagent/`, their private snippets/assets, and `cloud-connectors/` are not published here. `.mintignore` additionally excludes enterprise CLI and connector reference snippets that public code generators recreate. The common `odigos profile` command remains part of OSS documentation. New private content must be placed in the enterprise repo, not merely hidden from this site's navigation.

Run the existing CLI, CRD, RBAC, destination, and instrumentation generators as before. Generated files remain reproducible in this repo. Review corresponding updates to the enterprise repo's snapshot of `snippets/shared`; do not overwrite edition-specific prose wholesale. The enterprise README explains its build and release process.

## Publishing

Mintlify deploys the default branch and provides PR previews. **Deploy and verify the protected enterprise site first**, then merge this public split. `docs.json` redirects old enterprise routes to the same paths on `enterprise.docs.odigos.io`; existing OSS legacy redirects remain local. Public Git history and earlier copies of previously public docs are not erased by this move.

Run `node scripts/check-docs.mjs` and `npx mint@4.2.876 validate` before publishing. The source check verifies navigation, imports/assets, edition boundaries, and private publishing exclusions.
