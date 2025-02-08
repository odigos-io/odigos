import type { FetchedSource } from '@/types';
import { DISPLAY_TITLES } from '@odigos/ui-utils';
import { DataCardFieldsProps } from '@odigos/ui-components';

const buildCard = (source: FetchedSource) => {
  const { name, kind, namespace } = source;

  const arr: DataCardFieldsProps['data'] = [
    { title: DISPLAY_TITLES.NAMESPACE, value: namespace },
    { title: DISPLAY_TITLES.KIND, value: kind },
    { title: DISPLAY_TITLES.NAME, value: name, tooltip: 'K8s resource name' },
  ];

  return arr;
};

export default buildCard;
