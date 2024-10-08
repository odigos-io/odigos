import React, { useEffect, useMemo } from 'react';
import { CardDetails } from '@/components';
import { useDrawerStore } from '@/store';
import { K8sActualSource } from '@/types';

const SourceDrawer: React.FC = () => {
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);

  useEffect(() => {
    console.log({ selectedItem });
  }, [selectedItem]);

  const cardData = useMemo(() => {
    if (!selectedItem) return [];

    // Destructure necessary fields from the selected item
    const { name, kind, namespace, instrumentedApplicationDetails } =
      selectedItem.item as K8sActualSource;

    // Extract the first container and condition if available
    const container = instrumentedApplicationDetails?.containers?.[0];

    return [
      { title: 'Name', value: name || 'N/A' },
      { title: 'Kind', value: kind || 'N/A' },
      { title: 'Namespace', value: namespace || 'N/A' },
      { title: 'Container Name', value: container?.containerName || 'N/A' },
      { title: 'Language', value: container?.language || 'N/A' },
    ];
  }, [selectedItem]);

  return <CardDetails data={cardData} />;
};

export { SourceDrawer };
