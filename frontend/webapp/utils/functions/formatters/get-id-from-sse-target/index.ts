import { type WorkloadId } from '@/types';
import { Types } from '@odigos/ui-components';

export const getIdFromSseTarget = (target: string, type: Types.ENTITY_TYPES) => {
  switch (type) {
    case Types.ENTITY_TYPES.SOURCE: {
      const id: WorkloadId = {
        namespace: '',
        name: '',
        kind: '',
      };

      target.split('&').forEach((str) => {
        const [key, value] = str.split('=');
        id[key as keyof WorkloadId] = value;
      });

      return id;
    }

    default:
      return target as string;
  }
};
