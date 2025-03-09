import type { Source, WorkloadId } from '@odigos/ui-utils';

export const getWorkloadId = ({ namespace, name, kind }: Source): WorkloadId => {
  return { namespace, name, kind };
};
