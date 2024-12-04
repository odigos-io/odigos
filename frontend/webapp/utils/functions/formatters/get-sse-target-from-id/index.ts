import { OVERVIEW_ENTITY_TYPES, type WorkloadId } from '@/types';

export const getSseTargetFromId = (id: string | WorkloadId, type: OVERVIEW_ENTITY_TYPES) => {
  switch (type) {
    case OVERVIEW_ENTITY_TYPES.SOURCE: {
      let target = '';

      Object.entries(id as WorkloadId).forEach(([key, value]) => {
        target += `${key}=${value}&`;
      });

      target.slice(0, -1);

      return target;
    }

    default:
      return id as string;
  }
};
