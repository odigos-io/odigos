import { type K8sActualSource } from '@/types';
import { type WorkloadId } from '@odigos/ui-components';

const buildDrawerItem = (id: WorkloadId, formData: { otelServiceName: string }, drawerItem: K8sActualSource): K8sActualSource => {
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
