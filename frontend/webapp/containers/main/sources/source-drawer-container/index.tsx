import React, { useEffect, useMemo, useState } from 'react';
import styled from 'styled-components';
import { useSourceCRUD } from '@/hooks';
import { useDrawerStore } from '@/store';
import { CardDetails } from '@/components';
import type { K8sActualSource } from '@/types';
import { getMainContainerLanguageLogo } from '@/utils';
import { UpdateSourceBody } from '../update-source-body';
import OverviewDrawer from '../../overview/overview-drawer';

const EMPTY_FORM = {
  reportedName: '',
};

const SourceDrawer: React.FC = () => {
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);
  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);

  const [formData, setFormData] = useState({
    ...EMPTY_FORM,
  });

  const handleFormChange = (key: keyof typeof EMPTY_FORM, val: any) => {
    setFormData((prev) => ({
      ...prev,
      [key]: val,
    }));
  };

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

  useEffect(() => {
    if (!selectedItem || !isEditing) {
      setFormData({ ...EMPTY_FORM });
    } else {
      const { item } = selectedItem as { item: K8sActualSource };

      setFormData({
        reportedName: item.reportedName || '',
      });
    }
  }, [selectedItem, isEditing]);

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

  const handleSave = async () => {
    const { namespace, name, kind } = item as K8sActualSource;

    await updateSource({ namespace, kind, name }, formData);
  };

  return (
    <OverviewDrawer
      title={(item as K8sActualSource).reportedName || (item as K8sActualSource).name}
      titleTooltip={
        !(item as K8sActualSource).reportedName
          ? 'This is the default service name that runs in your cluster. You can override this name.'
          : 'This overrides the default service name that runs in your cluster.'
      }
      imageUri={getMainContainerLanguageLogo(item as K8sActualSource)}
      isEdit={isEditing}
      isFormDirty={isFormDirty}
      onEdit={handleEdit}
      onSave={handleSave}
      onDelete={handleDelete}
      onCancel={handleCancel}
    >
      {isEditing ? (
        <FormContainer>
          <UpdateSourceBody
            formData={formData}
            handleFormChange={(...params) => {
              setIsFormDirty(true);
              handleFormChange(...params);
            }}
          />
        </FormContainer>
      ) : (
        <CardDetails data={cardData} />
      )}
    </OverviewDrawer>
  );
};

export { SourceDrawer };

const FormContainer = styled.div`
  width: 100%;
  height: 100%;
  max-height: calc(100vh - 220px);
  overflow: overlay;
  overflow-y: auto;
`;
