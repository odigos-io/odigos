import { BACKEND_BOOLEAN } from '../constants';
import { STATUSES, type ActualDestination, type K8sActualSource } from '@/types';

export const getHealthStatus = (item: K8sActualSource | ActualDestination) => {
  const conditions = (item as K8sActualSource)?.instrumentedApplicationDetails?.conditions || (item as ActualDestination)?.conditions || [];
  const isUnhealthy = !conditions.length || !!conditions.find(({ status }) => status === BACKEND_BOOLEAN.FALSE);

  return isUnhealthy ? STATUSES.UNHEALTHY : STATUSES.HEALTHY;
};
