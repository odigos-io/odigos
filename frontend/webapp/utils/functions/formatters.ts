import { OVERVIEW_ENTITY_TYPES, WorkloadId } from '@/types';

export const formatBytes = (bytes?: number) => {
  if (!bytes) return '0 KB/s';

  const sizes = ['Bytes', 'KB/s', 'MB/s', 'GB/s', 'TB/s'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const value = bytes / Math.pow(1024, i);

  return `${value.toFixed(1)} ${sizes[i]}`;
};

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
        id[key] = value;
      });

      return id;
    }

    default:
      return target as string;
  }
};

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
