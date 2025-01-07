# Development

## Make changes to CRD

1. Update the relevant CRD `go` file in the `api/odigos` directory.
2. Run `make generate` to update the auto generated `zz_generated.deepcopy.go` file.
3. Run `make manifests` to update the CRD yaml files in the `config/crd/bases` directory.
4. Run `make generate-client` to update the auto generated files under `api/generated`.
5. Run `make sync-helm-crd` to sync the CRD yaml files to the helm chart.

you can run `make all` to combine steps 2-5 instead of running them individually.

## CRD versioning

Odigos CRDs are versioned. For example the `odigosconfigurations.odigos.io` resource version is `v1alpha1`.

The version should be bumped when a breaking change is made to a CRD. A breaking change is a change that requires a migration of the objects from the old version to the new one. 

**No** need to bump version:
- Removing a field from a CRD is not a breaking change.
- Adding a new optional field to a CRD (with a default value) is not a breaking change.

**Need** version bump:
- Adding a new required field to a CRD is a breaking change. existing objects should be migrated to populate this new field.
- Any change to the CRD semantics (e.g. changing the type of a field, changing the way values are used) is a breaking change and requires a new CRD version change and migration of the objects from old version to the new one.

