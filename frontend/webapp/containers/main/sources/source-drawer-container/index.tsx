import React, { useMemo, useState } from 'react';
import { useDrawerStore } from '@/store';
import { useActualSources } from '@/hooks';
import { CardDetails } from '@/components';
import OverviewDrawer from '../../overview/overview-drawer';
import { K8sActualSource, PatchSourceRequestInput, WorkloadId } from '@/types';
import { getMainContainerLanguageLogo } from '@/utils/constants/programming-languages';

const SourceDrawer: React.FC = () => {
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);
  const setSelectedItem = useDrawerStore(({ setSelectedItem }) => setSelectedItem);
  const [isEditing, setIsEditing] = useState(false);

  const { updateActualSource, deleteSourcesForNamespace } = useActualSources();

  const cardData = useMemo(() => {
    if (!selectedItem) return [];

    // Destructure necessary fields from the selected item
    const { name, kind, namespace, instrumentedApplicationDetails } = selectedItem.item as K8sActualSource;

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

  if (!selectedItem?.item) return null;
  const { item } = selectedItem;

  const handleEdit = (bool?: boolean) => {
    if (typeof bool === 'boolean') {
      setIsEditing(bool);
    } else {
      setIsEditing(true);
    }
  };

  const handleCancel = () => {
    setIsEditing(false);
  };

  const handleDelete = async () => {
    const { namespace, name, kind } = item as K8sActualSource;

    try {
      await deleteSourcesForNamespace(namespace, [
        {
          kind,
          name,
          selected: false,
        },
      ]);
      setSelectedItem(null);
    } catch (error) {
      console.error('Error deleting source:', error);
    }
  };

  const handleSave = async (newTitle: string) => {
    const { namespace, name, kind } = item as K8sActualSource;

    const sourceId: WorkloadId = {
      namespace,
      kind,
      name,
    };

    const patchRequest: PatchSourceRequestInput = {
      reportedName: newTitle,
    };

    try {
      await updateActualSource(sourceId, patchRequest);
      setSelectedItem(null);
    } catch (error) {
      console.error('Error updating source:', error);
    }
  };

  return (
    <OverviewDrawer
      title={(item as K8sActualSource).reportedName}
      imageUri={getMainContainerLanguageLogo(item as K8sActualSource)}
      isEdit={isEditing}
      clickEdit={handleEdit}
      clickSave={handleSave}
      clickDelete={handleDelete}
      clickCancel={handleCancel}
    >
      <CardDetails data={cardData} />
    </OverviewDrawer>
  );
};

export { SourceDrawer };
