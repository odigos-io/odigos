import type { K8sActualSource } from '@/types';

const buildCard = (source: K8sActualSource) => {
  const { name, kind, namespace, instrumentedApplicationDetails } = source;
  const { containerName, language } = instrumentedApplicationDetails?.containers?.[0] || {};

  const arr = [
    { title: 'Name', value: name || 'N/A' },
    { title: 'Kind', value: kind || 'N/A' },
    { title: 'Namespace', value: namespace || 'N/A' },
    { title: 'Container Name', value: containerName || 'N/A' },
    { title: 'Language', value: language || 'N/A' },
  ];

  return arr;
};

export default buildCard;
