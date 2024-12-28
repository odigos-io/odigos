import type { K8sActualSource, WorkloadId } from '@/types';

const buildDrawerItem = (id: WorkloadId, formData: { reportedName: string }, drawerItem: K8sActualSource): K8sActualSource => {
  const { namespace, name, kind } = id;
  const { reportedName } = formData;
  const { numberOfInstances, serviceName, conditions, containers, selected } = drawerItem;

  return {
    namespace,
    name,
    kind,
    numberOfInstances,
    serviceName,
    reportedName,
    conditions,
    containers,
    selected,
  };
};

export default buildDrawerItem;
