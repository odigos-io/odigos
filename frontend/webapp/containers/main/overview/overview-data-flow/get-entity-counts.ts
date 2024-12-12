import { type ComputePlatformMapped, OVERVIEW_ENTITY_TYPES } from '@/types';

interface Params {
  computePlatform?: ComputePlatformMapped['computePlatform'];
}

export type EntityCounts = Record<OVERVIEW_ENTITY_TYPES, number>;

export const getEntityCounts = ({ computePlatform }: Params) => {
  const unfilteredCounts: EntityCounts = {
    [OVERVIEW_ENTITY_TYPES.RULE]: computePlatform?.instrumentationRules.length || 0,
    [OVERVIEW_ENTITY_TYPES.SOURCE]: computePlatform?.k8sActualSources.length || 0,
    [OVERVIEW_ENTITY_TYPES.ACTION]: computePlatform?.actions.length || 0,
    [OVERVIEW_ENTITY_TYPES.DESTINATION]: computePlatform?.destinations.length || 0,
  };

  return unfilteredCounts;
};
