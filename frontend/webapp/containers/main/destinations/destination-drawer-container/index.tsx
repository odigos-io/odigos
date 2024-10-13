import React, { useMemo } from 'react';
import { useDrawerStore } from '@/store';
import { CardDetails } from '@/components';
import { ActualDestination } from '@/types';

const DestinationDrawer: React.FC = () => {
  const destination = useDrawerStore(({ selectedItem }) => selectedItem);
  const cardData = useMemo(() => {
    const { exportedSignals, destinationType } =
      destination?.item as ActualDestination;

    const monitors = Object.keys(exportedSignals)
      .map((key) => (exportedSignals[key] === true ? key : null))
      .filter(Boolean)
      .join(', ');

    return [
      { title: 'Destination', value: destinationType.displayName || 'N/A' },
      {
        title: 'Monitors',
        value: monitors,
      },
    ];
  }, [destination]);

  return <CardDetails data={cardData} />;
};

export { DestinationDrawer };
