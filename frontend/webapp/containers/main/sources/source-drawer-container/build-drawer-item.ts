import type { K8sActualSource, WorkloadId } from '@/types';

const buildDrawerItem = (id: WorkloadId, formData: { otelServerName: string }, drawerItem: K8sActualSource): K8sActualSource => {
  const { namespace, name, kind } = id;
  const { otelServerName } = formData;
  const { numberOfInstances, conditions, containers, selected } = drawerItem;

  return {
    namespace,
    name,
    kind,
    numberOfInstances,
    otelServerName,
    conditions,
    containers,
    selected,
  };
};

export default buildDrawerItem;
