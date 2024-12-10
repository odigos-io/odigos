import { DISPLAY_TITLES } from '@/utils';
import type { K8sActualSource } from '@/types';
import { DataCardRow } from '@/reuseable-components';

const buildCard = (source: K8sActualSource) => {
  const { name, kind, namespace } = source;

  const arr: DataCardRow[] = [
    { title: DISPLAY_TITLES.NAMESPACE, value: namespace },
    { title: DISPLAY_TITLES.KIND, value: kind },
    { title: DISPLAY_TITLES.NAME, value: name, tooltip: 'K8s resource name' },
  ];

  return arr;
};

export default buildCard;
