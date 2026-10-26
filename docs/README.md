# Odigos Documentation

Source for the public Open Source documentation at [docs.odigos.io](https://docs.odigos.io). This site documents the Kubernetes Agent Open Source edition, plus overview pages for Odigos Central, the VM Agent, Cloud Connectors, the Enterprise MCP server, and continuous profiling. The full enterprise documentation is published from the odigos-enterprise repository at [enterprise.docs.odigos.io](https://enterprise.docs.odigos.io).

### Development

The documentation is built with [Mintlify](https://www.npmjs.com/package/mintlify). To preview changes locally:

```
# make sure you're in `/docs` folder, where `docs.json` and `package.json` files are
# install the mintlify npm package (if you don't already have it)
npm i -g mint
mint dev
```

Generated CLI, CRD, destination, instrumentation, and RBAC snippets are written under `snippets/oss`:

```
make cli-docs
make rbac-docs
cd api && make docgen
cd docs && python sync-dest-doc.py
```

`sync-dest-doc.py` updates Open Source destination pages and skips Pyroscope, which is documented only on the enterprise site. Copy changed generated snippets into the enterprise repository's `snippets/enterprise` tree when those pages should stay in sync.

### Publishing Changes

Changes will be deployed to production automatically after pushing to the default branch.

You can also preview changes using PRs, which generates a preview link of the docs.
