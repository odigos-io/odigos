import React, { useEffect, useMemo, useState } from 'react';
import buildCard from './build-card';
import styled from 'styled-components';
import { type FetchedSource } from '@/types';
import buildDrawerItem from './build-drawer-item';
import { CodeIcon, ListIcon } from '@odigos/ui-icons';
import { useDrawerStore } from '@odigos/ui-containers';
import { UpdateSourceBody } from '../update-source-body';
import { useDescribeSource, useSourceCRUD } from '@/hooks';
import OverviewDrawer from '../../overview/overview-drawer';
import { ACTION, DATA_CARDS } from '@/utils';
import { CONDITION_STATUS, ENTITY_TYPES, getEntityIcon, safeJsonStringify, type WorkloadId } from '@odigos/ui-utils';
import { ConditionDetails, DATA_CARD_FIELD_TYPES, DataCard, type DataCardFieldsProps, Segment } from '@odigos/ui-components';

interface Props {}

const EMPTY_FORM = {
  otelServiceName: '',
};

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

export const SourceDrawer: React.FC<Props> = () => {
  const { selectedItem, setSelectedItem } = useDrawerStore();
  const { sources, persistSources, updateSource } = useSourceCRUD({
    onSuccess: (type) => {
      setIsEditing(false);
      setIsFormDirty(false);

      if (type === ACTION.DELETE) setSelectedItem(null);
      else reSelectItem();
    },
  });

  // TODO: check if the item is already set on-mount
  // drawerItem['item'] = sources.find(
  //   (item) => item.kind === (drawerItem['id'] as WorkloadId).kind && item.name === (drawerItem['id'] as WorkloadId).name && item.namespace === (drawerItem['id'] as WorkloadId).namespace,
  // );
  const reSelectItem = (fetchedItems?: typeof sources) => {
    const { item } = selectedItem as { item: FetchedSource };
    const { namespace, name, kind } = item;
    const id = { namespace, name, kind };

    if (!!fetchedItems?.length) {
      const found = fetchedItems.find((x) => x.namespace === namespace && x.name === name && x.kind === kind);
      if (!!found) {
        return setSelectedItem({ id, type: ENTITY_TYPES.SOURCE, item: found });
      }
    }

    setSelectedItem({ id, type: ENTITY_TYPES.SOURCE, item: buildDrawerItem(id, formData, item) });
  };

  // This should keep the drawer up-to-date with the latest data
  useEffect(() => reSelectItem(sources), [sources]);

  const [isPrettyMode, setIsPrettyMode] = useState(true); // for "describe source"
  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);
  const [formData, setFormData] = useState({ ...EMPTY_FORM });

  const handleFormChange = (key: keyof typeof EMPTY_FORM, val: any) => setFormData((prev) => ({ ...prev, [key]: val }));
  const resetFormData = () => setFormData({ ...EMPTY_FORM });

  useEffect(() => {
    if (!selectedItem || !isEditing) {
      resetFormData();
    } else {
      const { item } = selectedItem as { item: FetchedSource };
      handleFormChange('otelServiceName', item.otelServiceName || item.name || '');
    }
  }, [selectedItem, isEditing]);

  const cardData = useMemo(() => {
    if (!selectedItem) return [];

    const { item } = selectedItem as { item: FetchedSource };
    const arr = buildCard(item);

    return arr;
  }, [selectedItem]);

  const containersData = useMemo(() => {
    if (!selectedItem) return [];

    const { item } = selectedItem as { item: FetchedSource };
    const hasPresenceOfOtherAgent =
      item?.conditions?.some((condition) => condition.status === CONDITION_STATUS.FALSE && condition.message.includes('device not added to any container due to the presence of another agent')) ||
      false;

    return (
      item?.containers?.map(
        (container) =>
          ({
            type: DATA_CARD_FIELD_TYPES.SOURCE_CONTAINER,
            width: '100%',
            value: JSON.stringify({
              ...container,
              hasPresenceOfOtherAgent,
            }),
          } as DataCardFieldsProps['data'][0]),
      ) || []
    );
  }, [selectedItem]);

  if (!selectedItem?.item) return null;
  const { id, item } = selectedItem as { id: WorkloadId; item: FetchedSource };
  const { data: describe, restructureForPrettyMode } = useDescribeSource(id);

  const handleEdit = (bool?: boolean) => {
    setIsEditing(typeof bool === 'boolean' ? bool : true);
  };

  const handleCancel = () => {
    setIsEditing(false);
    setIsFormDirty(false);
  };

  const handleDelete = async () => {
    const { namespace } = item;

    await persistSources({ [namespace]: [{ ...item, selected: false }] }, {});
  };

  const handleSave = async () => {
    const title = formData.otelServiceName !== item.name ? formData.otelServiceName : '';
    handleFormChange('otelServiceName', title);
    await updateSource(id, { ...formData, otelServiceName: title });
  };

  return (
    <OverviewDrawer
      title={item.otelServiceName || item.name}
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
          <ConditionDetails conditions={item.conditions || []} />
          <DataCard title={DATA_CARDS.SOURCE_DETAILS} data={cardData} />
          <DataCard title={DATA_CARDS.DETECTED_CONTAINERS} titleBadge={containersData.length} description={DATA_CARDS.DETECTED_CONTAINERS_DESCRIPTION} data={containersData} />
          <DataCard
            title={DATA_CARDS.DESCRIBE_SOURCE}
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
