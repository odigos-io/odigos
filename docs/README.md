# Odigos Documentation

The content and configuration powering the Odigos documentation is available at [docs.odigos.io](https://docs.odigos.io)

### 🚀 Setup

Simply merge in this PR and your documentation will be connected!

### 👩‍💻 Development

The documentation is build on [Mintlify](https://www.npmjs.com/package/mintlify). To preview the documentation changes locally:

```
# make sure you're in `/docs` folder, where `docs.json` and `package.json` files are
# install the mintlify npm package (if you don't already have it)
npm i -g mint
mint dev
```

### 😎 Publishing Changes

Changes will be deployed to production automatically after pushing to the default branch.

You can also preview changes using PRs, which generates a preview link of the docs.

## Protected documentation site

The password-protected site at [enterprise.docs.odigos.io](https://enterprise.docs.odigos.io) is maintained in `odigos-enterprise/docs`. Both repositories currently retain the complete documentation set. Page moves, removals, and edition-specific navigation changes will be handled in separate migration PRs. This public site keeps its existing pages and redirects.

Run `node scripts/check-docs.mjs` to validate navigation, imports, and image references. The check permits both editions and does not impose page ownership.
