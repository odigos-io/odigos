import {  type Source, type Workload } from '@odigos/ui-kit/types';

export type WorkloadWithOdigosHealthStatus = Workload & {
    workloadOdigosHealthStatus?: Source['workloadOdigosHealthStatus'];
    rollbackOccurred?: boolean;
  };