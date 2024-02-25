import { NewActionCard } from '@/components';
import React from 'react';

const ITEMS = [
  {
    id: '1',
    title: 'Cluster Attributes',
    description:
      'With cluster attributes, you can define the attributes of the cluster. This is useful for filtering and grouping spans in your backend.',
    type: 'cluster-attributes',
    icon: 'cluster-attributes',
  },
  {
    id: '2',
    title: 'Filter',
    description: 'Filter spans based on the attributes of the span.',
    type: 'filter',
    icon: 'filter',
  },
];

export function ChooseActionContainer() {
  function renderActionsList() {
    return ITEMS.map((item) => {
      return (
        <div key={item.id}>
          <NewActionCard item={item} />
        </div>
      );
    });
  }

  return <>{renderActionsList()}</>;
}
