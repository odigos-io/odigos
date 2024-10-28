import React, { useMemo } from 'react';
import { useDrawerStore } from '@/store';
import { CardDetails } from '@/components';
import type { ActionDataParsed } from '@/types';
import buildCardFromActionSpec from './build-card-from-action-spec';

interface Props {
  isEditing: boolean;
}

const ActionDrawer: React.FC = ({ isEditing }: Props) => {
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);

  const cardData = useMemo(() => {
    if (!selectedItem) return [];

    const arr = buildCardFromActionSpec(selectedItem.item as ActionDataParsed);

    return arr;
  }, [selectedItem]);

  return <CardDetails data={cardData} />;
};

export { ActionDrawer };
