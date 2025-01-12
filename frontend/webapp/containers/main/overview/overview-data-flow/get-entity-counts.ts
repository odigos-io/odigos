import { OVERVIEW_ENTITY_TYPES } from '@/types';

interface Params {
  sources?: any[];
  destinations?: any[];
  actions?: any[];
  instrumentationRules?: any[];
}

export type EntityCounts = Record<OVERVIEW_ENTITY_TYPES, number>;

export const getEntityCounts = ({ sources, destinations, actions, instrumentationRules }: Params) => {
  const unfilteredCounts: EntityCounts = {
    [OVERVIEW_ENTITY_TYPES.SOURCE]: sources?.length || 0,
    [OVERVIEW_ENTITY_TYPES.DESTINATION]: destinations?.length || 0,
    [OVERVIEW_ENTITY_TYPES.ACTION]: actions?.length || 0,
    [OVERVIEW_ENTITY_TYPES.RULE]: instrumentationRules?.length || 0,
  };

  return unfilteredCounts;
};
