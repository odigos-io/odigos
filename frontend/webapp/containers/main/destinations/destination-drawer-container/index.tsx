import React, { useMemo, useState } from 'react';
import styled from 'styled-components';
import { safeJsonParse } from '@/utils';
import { useDrawerStore } from '@/store';
import { CardDetails } from '@/components';
import type { ActualDestination } from '@/types';
import OverviewDrawer from '../../overview/overview-drawer';
import { useDestinationCRUD, useDestinationFormData, useDestinationTypes } from '@/hooks';
import { ConnectDestinationModalBody } from '../add-destination/connect-destination-modal-body';

interface Props {}

const DestinationDrawer: React.FC<Props> = () => {
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);
  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);

  const { updateDestination, deleteDestination } = useDestinationCRUD();
  const { formData, handleFormChange, resetFormData, validateForm, loadFormWithDrawerItem, destinationTypeDetails, dynamicFields, setDynamicFields } = useDestinationFormData({
    destinationType: (selectedItem?.item as ActualDestination)?.destinationType?.type,
    // supportedSignals: thisDestination?.supportedSignals,
    preLoadedFields: (selectedItem?.item as ActualDestination)?.fields,
  });

  const cardData = useMemo(() => {
    if (!selectedItem) return [];

    const buildMonitorsList = (exportedSignals: ActualDestination['exportedSignals']): string => {
      return (
        Object.keys(exportedSignals)
          .filter((key) => exportedSignals[key] && key !== '__typename')
          .join(', ') || 'None'
      );
    };

    const buildDestinationFieldData = (parsedFields: Record<string, string>) => {
      return Object.entries(parsedFields).map(([key, value]) => {
        const found = destinationTypeDetails?.fields?.find((field) => field.name === key);

        const { type } = safeJsonParse(found?.componentProperties, { type: '' });
        const secret = type === 'password' ? new Array(value.length).fill('•').join('') : '';

        return {
          title: found?.displayName || key,
          value: secret || value || 'N/A',
        };
      });
    };

    const { exportedSignals, destinationType, fields } = selectedItem.item as ActualDestination;
    const parsedFields = safeJsonParse<Record<string, string>>(fields, {});
    const fieldsData = buildDestinationFieldData(parsedFields);

    return [{ title: 'Destination', value: destinationType.displayName || 'N/A' }, { title: 'Monitors', value: buildMonitorsList(exportedSignals) }, ...fieldsData];
  }, [selectedItem, destinationTypeDetails]);

  const { destinations } = useDestinationTypes();
  const thisDestination = useMemo(() => {
    if (!destinations.length || !selectedItem || !isEditing) {
      resetFormData();
      return undefined;
    }

    const { item } = selectedItem as { item: ActualDestination };
    const found = destinations.map(({ items }) => items.filter(({ type }) => type === item.destinationType.type)).filter((arr) => !!arr.length)[0][0];

    if (!found) return undefined;

    loadFormWithDrawerItem(selectedItem);

    return found;
  }, [destinations, selectedItem, isEditing]);

  if (!selectedItem?.item) return null;
  const { id, item } = selectedItem;

  const handleEdit = (bool?: boolean) => {
    if (typeof bool === 'boolean') {
      setIsEditing(bool);
    } else {
      setIsEditing(true);
    }
  };

  const handleCancel = () => {
    resetFormData();
    setIsEditing(false);
  };

  const handleDelete = async () => {
    await deleteDestination(id as string);
  };

  const handleSave = async (newTitle: string) => {
    if (validateForm({ withAlert: true })) {
      const title = newTitle !== (item as ActualDestination).destinationType.displayName ? newTitle : '';

      await updateDestination(id as string, { ...formData, name: title });
    }
  };

  return (
    <OverviewDrawer
      title={(item as ActualDestination).name || (item as ActualDestination).destinationType.displayName}
      imageUri={(item as ActualDestination).destinationType.imageUrl}
      isEdit={isEditing}
      isFormDirty={isFormDirty}
      onEdit={handleEdit}
      onSave={handleSave}
      onDelete={handleDelete}
      onCancel={handleCancel}
    >
      {isEditing ? (
        <FormContainer>
          <ConnectDestinationModalBody
            isUpdate
            destination={thisDestination}
            formData={formData}
            handleFormChange={(...params) => {
              setIsFormDirty(true);
              handleFormChange(...params);
            }}
            dynamicFields={dynamicFields}
            setDynamicFields={(...params) => {
              setIsFormDirty(true);
              setDynamicFields(...params);
            }}
          />
        </FormContainer>
      ) : (
        <CardDetails data={cardData} />
      )}
    </OverviewDrawer>
  );
};

export { DestinationDrawer };

const FormContainer = styled.div`
  width: 100%;
  height: 100%;
  max-height: calc(100vh - 220px);
  overflow: overlay;
  overflow-y: auto;
`;
