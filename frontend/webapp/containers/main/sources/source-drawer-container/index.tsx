import React, { useMemo, useState } from 'react';
import { useDrawerStore } from '@/store';
import { useSourceCRUD } from '@/hooks';
import { CardDetails } from '@/components';
import OverviewDrawer from '../../overview/overview-drawer';
import { K8sActualSource, PatchSourceRequestInput, WorkloadId } from '@/types';
import { getMainContainerLanguageLogo } from '@/utils/constants/programming-languages';

const SourceDrawer: React.FC = () => {
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);
  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);

  const { deleteSources, updateSource } = useSourceCRUD();

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
    const { namespace } = item as K8sActualSource;

    await deleteSources({ [namespace]: [item as K8sActualSource] });
  };

  const handleSave = async (newTitle: string) => {
    const { namespace, name, kind } = item as K8sActualSource;

    await updateSource({ namespace, kind, name }, { reportedName: newTitle });
  };

  return (
    <OverviewDrawer
      title={(item as K8sActualSource).reportedName}
      imageUri={getMainContainerLanguageLogo(item as K8sActualSource)}
      isEdit={isEditing}
      isFormDirty={isFormDirty}
      onEdit={handleEdit}
      onSave={handleSave}
      onDelete={handleDelete}
      onCancel={handleCancel}
    >
      <CardDetails data={cardData} />
    </OverviewDrawer>
  );
};

export { SourceDrawer };
