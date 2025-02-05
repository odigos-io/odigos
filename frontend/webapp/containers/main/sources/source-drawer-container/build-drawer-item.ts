import { type FetchedSource } from '@/types';
import { type WorkloadId } from '@odigos/ui-utils';

const buildDrawerItem = (id: WorkloadId, formData: { otelServiceName: string }, drawerItem: FetchedSource): FetchedSource => {
  const { namespace, name, kind } = id;
  const { otelServiceName } = formData;
  const { numberOfInstances, conditions, containers, selected } = drawerItem;

  return {
    namespace,
    name,
    kind,
    numberOfInstances,
    otelServiceName,
    conditions,
    containers,
    selected,
  };
};

export default buildDrawerItem;
