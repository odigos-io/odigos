import React, { useMemo, useState } from 'react';
import buildCard from './build-card';
import styled from 'styled-components';
import { CodeIcon, ListIcon } from '@odigos/ui-icons';
import { useDrawerStore } from '@odigos/ui-containers';
import { UpdateSourceBody } from '../update-source-body';
import { useDescribeSource, useSourceCRUD } from '@/hooks';
import { CRUD, DISPLAY_TITLES, ENTITY_TYPES, getEntityIcon, safeJsonStringify, type WorkloadId } from '@odigos/ui-utils';
import { ConditionDetails, DATA_CARD_FIELD_TYPES, DataCard, type DataCardFieldsProps, Segment } from '@odigos/ui-components';
import OverviewDrawer from '../../overview/overview-drawer';

interface Props {}

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

const EMPTY_FORM = {
  otelServiceName: '',
};

export const SourceDrawer: React.FC<Props> = () => {
  const { drawerEntityId, setDrawerEntityId, setDrawerType } = useDrawerStore();

  const [isPrettyMode, setIsPrettyMode] = useState(true); // for "describe source"
  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);

  const [formData, setFormData] = useState({ ...EMPTY_FORM });
  const handleFormChange = (key: keyof typeof EMPTY_FORM, val: any) => setFormData((prev) => ({ ...prev, [key]: val }));
  const resetFormData = () => setFormData({ ...EMPTY_FORM });

  const { data: describe, restructureForPrettyMode } = useDescribeSource(drawerEntityId as WorkloadId);
  const { sources, persistSources, updateSource } = useSourceCRUD({
    onSuccess: (type) => {
      setIsEditing(false);
      setIsFormDirty(false);

      if (type === CRUD.DELETE) {
        setDrawerType(null);
        setDrawerEntityId(null);
        resetFormData();
      }
    },
  });

  const thisItem = useMemo(() => {
    const found = sources?.find((x) => x.namespace === (drawerEntityId as WorkloadId).namespace && x.name === (drawerEntityId as WorkloadId).name && x.kind === (drawerEntityId as WorkloadId).kind);
    if (!!found) handleFormChange('otelServiceName', found.otelServiceName || found.name || '');

    return found;
  }, [sources, drawerEntityId]);

  if (!thisItem) return null;

  const containersData =
    thisItem.containers?.map(
      (container) =>
        ({
          type: DATA_CARD_FIELD_TYPES.SOURCE_CONTAINER,
          width: '100%',
          value: JSON.stringify(container),
        } as DataCardFieldsProps['data'][0]),
    ) || [];

  const handleEdit = (bool?: boolean) => {
    setIsEditing(typeof bool === 'boolean' ? bool : true);
  };

  const handleCancel = () => {
    setIsEditing(false);
    setIsFormDirty(false);
    handleFormChange('otelServiceName', thisItem.otelServiceName || thisItem.name || '');
  };

  const handleDelete = async () => {
    const { namespace } = thisItem;
    await persistSources({ [namespace]: [{ ...thisItem, selected: false }] }, {});
  };

  const handleSave = async () => {
    const title = formData.otelServiceName !== thisItem.name ? formData.otelServiceName : '';
    handleFormChange('otelServiceName', title);
    await updateSource(drawerEntityId as WorkloadId, { ...formData, otelServiceName: title });
  };

  return (
    <OverviewDrawer
      title={thisItem.otelServiceName || thisItem.name}
      titleTooltip='This attribute is used to identify the name of the service (service.name) that is generating telemetry data.'
      icon={getEntityIcon(ENTITY_TYPES.SOURCE)}
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
          <ConditionDetails conditions={thisItem.conditions || []} />
          <DataCard title={DISPLAY_TITLES.SOURCE_DETAILS} data={!!thisItem ? buildCard(thisItem) : []} />
          <DataCard title={DISPLAY_TITLES.DETECTED_CONTAINERS} titleBadge={containersData.length} description={DISPLAY_TITLES.DETECTED_CONTAINERS_DESCRIPTION} data={containersData} />
          <DataCard
            title={DISPLAY_TITLES.DESCRIBE_SOURCE}
            action={
              <Segment
                options={[
                  { icon: ListIcon, value: true },
                  { icon: CodeIcon, value: false },
                ]}
                selected={isPrettyMode}
                setSelected={setIsPrettyMode}
              />
            }
            data={[
              {
                type: DATA_CARD_FIELD_TYPES.CODE,
                value: JSON.stringify({
                  language: 'json',
                  code: safeJsonStringify(isPrettyMode ? restructureForPrettyMode(describe) : describe),
                  pretty: isPrettyMode,
                }),
                width: 'inherit',
              },
            ]}
          />
        </DataContainer>
      )}
    </OverviewDrawer>
  );
};
