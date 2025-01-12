import { OVERVIEW_ENTITY_TYPES, type WorkloadId } from '@/types';

export const getIdFromSseTarget = (target: string, type: OVERVIEW_ENTITY_TYPES) => {
  switch (type) {
    case OVERVIEW_ENTITY_TYPES.SOURCE: {
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
