import { DataCardRow } from '@/reuseable-components';
import type { K8sActualSource } from '@/types';

const buildCard = (source: K8sActualSource) => {
  const { name, kind, namespace, instrumentedApplicationDetails } = source;
  const { containerName, language } = instrumentedApplicationDetails?.containers?.[0] || {};

  const arr: DataCardRow[] = [
    { title: 'Namespace', value: namespace },
    { title: 'Kind', value: kind },
    { title: 'Container Name', value: containerName },
    { title: 'Name', value: name },
    { title: 'Language', value: language },
  ];

  return arr;
};

export default buildCard;
