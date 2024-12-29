import type { K8sActualSource, WorkloadId } from '@/types';

const buildDrawerItem = (id: WorkloadId, formData: { reportedName: string }, drawerItem: K8sActualSource): K8sActualSource => {
  const { namespace, name, kind } = id;
  const { reportedName } = formData;
  const { numberOfInstances, conditions, containers, selected } = drawerItem;

  return {
    namespace,
    name,
    kind,
    numberOfInstances,
    reportedName,
    conditions,
    containers,
    selected,
  };
};

export default buildDrawerItem;
