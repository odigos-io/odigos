# Odigos Documentation

The content and configuration powering the Odigos documentation is available at [docs.odigos.io](https://docs.odigos.io)

### 🚀 Setup

Simply merge in this PR and your documentation will be connected!

### 👩‍💻 Development

The documentation is build on [Mintlify](https://www.npmjs.com/package/mintlify). To preview the documentation changes locally:

```
# make sure you're in `/docs` folder, where `mint.json` and `package.json` files are
npm ci
npm run dev
```

### 😎 Publishing Changes

Changes will be deployed to production automatically after pushing to the default branch.

You can also preview changes using PRs, which generates a preview link of the docs.

#### Troubleshooting

- Mintlify dev isn't running - Run `mintlify install` it'll re-install dependencies.
- Mintlify dev is updating really slowly - Run `mintlify clear` to clear the cache.
