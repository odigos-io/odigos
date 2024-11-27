import type { K8sActualSource } from '@/types';

const buildCard = (source: K8sActualSource) => {
  const { name, kind, namespace, instrumentedApplicationDetails } = source;
  const { containerName, language } = instrumentedApplicationDetails?.containers?.[0] || {};

  const arr = [
    { title: 'Namespace', value: namespace },
    { title: 'Kind', value: kind },
    { title: 'Name', value: name },
    { title: 'Container Name', value: containerName || 'N/A' },
    { title: 'Language', value: language || 'N/A' },
  ];

  return arr;
};

export default buildCard;
