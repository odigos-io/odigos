import React, { useEffect, useMemo, useState } from 'react';
import buildCard from './build-card';
import styled from 'styled-components';
import { useSourceCRUD } from '@/hooks';
import { useDrawerStore } from '@/store';
import { CardDetails } from '@/components';
import buildDrawerItem from './build-drawer-item';
import { UpdateSourceBody } from '../update-source-body';
import OverviewDrawer from '../../overview/overview-drawer';
import { ACTION, getMainContainerLanguageLogo } from '@/utils';
import { OVERVIEW_ENTITY_TYPES, type K8sActualSource } from '@/types';

interface Props {}

const EMPTY_FORM = {
  reportedName: '',
};

const SourceDrawer: React.FC<Props> = () => {
  const { selectedItem, setSelectedItem } = useDrawerStore();

  const { deleteSources, updateSource } = useSourceCRUD({
    onSuccess: (type) => {
      setIsEditing(false);
      setIsFormDirty(false);

      if (type === ACTION.DELETE) {
        setSelectedItem(null);
      } else {
        const { item } = selectedItem as { item: K8sActualSource };
        const { namespace, name, kind } = item;
        const id = { namespace, name, kind };
        setSelectedItem({ id, type: OVERVIEW_ENTITY_TYPES.SOURCE, item: buildDrawerItem(id, formData, item) });
      }
    },
  });

  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);
  const [formData, setFormData] = useState({ ...EMPTY_FORM });

  const handleFormChange = (key: keyof typeof EMPTY_FORM, val: any) => setFormData((prev) => ({ ...prev, [key]: val }));
  const resetFormData = () => setFormData({ ...EMPTY_FORM });

  useEffect(() => {
    if (!selectedItem || !isEditing) {
      resetFormData();
    } else {
      const { item } = selectedItem as { item: K8sActualSource };
      handleFormChange('reportedName', item.reportedName || item.name || '');
    }
  }, [selectedItem, isEditing]);

  const cardData = useMemo(() => {
    if (!selectedItem) return [];

    const { item } = selectedItem as { item: K8sActualSource };
    const arr = buildCard(item);

    return arr;
  }, [selectedItem]);

  if (!selectedItem?.item) return null;
  const { item } = selectedItem;

  const handleEdit = (bool?: boolean) => {
    setIsEditing(typeof bool === 'boolean' ? bool : true);
  };

  const handleCancel = () => {
    setIsEditing(false);
    setIsFormDirty(false);
  };

  const handleDelete = async () => {
    const { namespace } = item as K8sActualSource;
    await deleteSources({ [namespace]: [item as K8sActualSource] });
  };

  const handleSave = async () => {
    const { namespace, name, kind } = item as K8sActualSource;
    const title = formData.reportedName !== (item as K8sActualSource).name ? formData.reportedName : '';
    await updateSource({ namespace, kind, name }, { ...formData, reportedName: title });
  };

  return (
    <OverviewDrawer
      title={(item as K8sActualSource).reportedName || (item as K8sActualSource).name}
      titleTooltip='This attribute is used to identify the name of the service (service.name) that is generating telemetry data.'
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
        <DataContainer>
          <CardDetails data={cardData} />
          {/* <CardDetails
            title='Resource Attributes'
            data={[
              { title: 'Service Name', tooltip: 'This overrides the default service name that runs in your cluster.', value: (item as K8sActualSource).reportedName || (item as K8sActualSource).name },
            ]}
          /> */}
        </DataContainer>
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

const DataContainer = styled.div`
  display: flex;
  flex-direction: column;
  gap: 12px;
`;
