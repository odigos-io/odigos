import { PlatformType } from '@odigos/ui-kit/types';

/**
 * GraphQL `ComputePlatformType` serializes as uppercase (`K8S` / `VM`).
 * ui-kit `PlatformType` values are lowercase (`k8s` / `vm` / `connector`).
 * Normalize at the GQL boundary so `isK8s` / `isVm` comparisons work.
 */
export const toPlatformType = (value?: string | null): PlatformType => {
  switch (value?.toLowerCase()) {
    case PlatformType.Vm:
      return PlatformType.Vm;
    case PlatformType.Connector:
      return PlatformType.Connector;
    default:
      return PlatformType.K8s;
  }
};
