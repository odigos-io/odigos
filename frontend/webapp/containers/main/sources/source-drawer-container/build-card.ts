import { DISPLAY_TITLES } from '@/utils';
import type { K8sActualSource } from '@/types';
import { DataCardRow } from '@/reuseable-components';

const buildCard = (source: K8sActualSource) => {
  const { name, kind, namespace, instrumentedApplicationDetails } = source;
  const { containerName, language } = instrumentedApplicationDetails?.containers?.[0] || {};

  const arr: DataCardRow[] = [
    { title: DISPLAY_TITLES.NAMESPACE, value: namespace },
    { title: DISPLAY_TITLES.KIND, value: kind },
    { title: DISPLAY_TITLES.CONTAINER_NAME, value: containerName },
    { title: DISPLAY_TITLES.NAME, value: name },
    { title: DISPLAY_TITLES.LANGUAGE, value: language },
  ];

  return arr;
};

export default buildCard;
