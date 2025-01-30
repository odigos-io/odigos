import { type WorkloadId } from '@/types';
import { Types } from '@odigos/ui-components';

export const getSseTargetFromId = (id: string | WorkloadId, type: Types.ENTITY_TYPES) => {
  switch (type) {
    case Types.ENTITY_TYPES.SOURCE: {
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
