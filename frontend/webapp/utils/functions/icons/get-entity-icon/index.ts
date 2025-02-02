import { OVERVIEW_ENTITY_TYPES } from '@/types';
import { ActionsIcon, DestinationsIcon, RulesIcon, SourcesIcon, Types } from '@odigos/ui-components';

export const getEntityIcon = (type: OVERVIEW_ENTITY_TYPES) => {
  const LOGOS: Record<OVERVIEW_ENTITY_TYPES, Types.SVG> = {
    [OVERVIEW_ENTITY_TYPES.ACTION]: ActionsIcon,
    [OVERVIEW_ENTITY_TYPES.DESTINATION]: DestinationsIcon,
    [OVERVIEW_ENTITY_TYPES.RULE]: RulesIcon,
    [OVERVIEW_ENTITY_TYPES.SOURCE]: SourcesIcon,
  };

  return LOGOS[type];
};
