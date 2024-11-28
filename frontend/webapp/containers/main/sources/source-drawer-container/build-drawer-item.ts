import type { K8sActualSource, WorkloadId } from '@/types';

const buildDrawerItem = (id: WorkloadId, formData: { reportedName: string }, drawerItem: K8sActualSource): K8sActualSource => {
  const { namespace, name, kind } = id;
  const { reportedName } = formData;
  const { selected, numberOfInstances, instrumentedApplicationDetails } = drawerItem;

  return {
    namespace,
    name,
    kind,
    reportedName,
    selected,
    numberOfInstances,
    instrumentedApplicationDetails,
  };
};

export default buildDrawerItem;
