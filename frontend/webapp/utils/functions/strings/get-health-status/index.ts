import { BACKEND_BOOLEAN } from '@/utils';
import { STATUSES, type ActualDestination, type K8sActualSource } from '@/types';

export const getHealthStatus = (item: K8sActualSource | ActualDestination) => {
  const conditions = item?.conditions || [];
  const isUnhealthy = !!conditions.find(({ status }) => status === BACKEND_BOOLEAN.FALSE);

  return isUnhealthy ? STATUSES.UNHEALTHY : STATUSES.HEALTHY;
};
