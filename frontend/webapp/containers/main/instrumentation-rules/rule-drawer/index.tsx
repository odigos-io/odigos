import React, { useEffect, useMemo, useState } from 'react';
import { RuleFormBody } from '../';
import buildCard from './build-card';
import styled from 'styled-components';
import { DataCard } from '@odigos/ui-components';
import buildDrawerItem from './build-drawer-item';
import { ACTION, DATA_CARDS, FORM_ALERTS } from '@/utils';
import OverviewDrawer from '../../overview/overview-drawer';
import { type InstrumentationRuleSpecMapped } from '@/types';
import { useDrawerStore, useNotificationStore } from '@odigos/ui-containers';
import { useInstrumentationRuleCRUD, useInstrumentationRuleFormData } from '@/hooks';
import { ENTITY_TYPES, getInstrumentationRuleIcon, INSTRUMENTATION_RULE_OPTIONS, NOTIFICATION_TYPE } from '@odigos/ui-utils';

interface Props {}

const FormContainer = styled.div`
  width: 100%;
  height: 100%;
  max-height: calc(100vh - 220px);
  overflow: overlay;
  overflow-y: auto;
`;

export const RuleDrawer: React.FC<Props> = () => {
  const { addNotification } = useNotificationStore();
  const { selectedItem, setSelectedItem } = useDrawerStore();
  const { formData, formErrors, handleFormChange, resetFormData, validateForm, loadFormWithDrawerItem } = useInstrumentationRuleFormData();
  const { instrumentationRules, updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD({
    onSuccess: (type) => {
      setIsEditing(false);
      setIsFormDirty(false);

      if (type === ACTION.DELETE) setSelectedItem(null);
      else reSelectItem();
    },
  });

  const reSelectItem = (fetchedItems?: typeof instrumentationRules) => {
    const { item } = selectedItem as { item: InstrumentationRuleSpecMapped };
    const { ruleId: id } = item;

    if (!!fetchedItems?.length) {
      const found = fetchedItems.find((x) => x.ruleId === id);
      if (!!found) {
        return setSelectedItem({ id, type: ENTITY_TYPES.INSTRUMENTATION_RULE, item: found });
      }
    }

    setSelectedItem({ id, type: ENTITY_TYPES.INSTRUMENTATION_RULE, item: buildDrawerItem(id, formData, item) });
  };

  // This should keep the drawer up-to-date with the latest data
  useEffect(() => reSelectItem(instrumentationRules), [instrumentationRules]);

  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);

  const cardData = useMemo(() => {
    if (!selectedItem) return [];

    const { item } = selectedItem as { item: InstrumentationRuleSpecMapped };
    const arr = buildCard(item);

    return arr;
  }, [selectedItem]);

  const thisRule = useMemo(() => {
    if (!selectedItem || !isEditing) {
      resetFormData();
      return undefined;
    }

    const { item } = selectedItem as { item: InstrumentationRuleSpecMapped };
    const found = INSTRUMENTATION_RULE_OPTIONS.find(({ type }) => type === item.type);

    loadFormWithDrawerItem(selectedItem);

    return found;
  }, [selectedItem, isEditing]);

  if (!selectedItem?.item) return null;
  const { id, item } = selectedItem as { id: string; item: InstrumentationRuleSpecMapped };

  const handleEdit = (bool?: boolean) => {
    if (!item.mutable && (bool || bool === undefined)) {
      addNotification({
        type: NOTIFICATION_TYPE.WARNING,
        title: FORM_ALERTS.FORBIDDEN,
        message: FORM_ALERTS.CANNOT_EDIT_RULE,
        crdType: ENTITY_TYPES.INSTRUMENTATION_RULE,
        target: id,
        hideFromHistory: true,
      });
    } else {
      setIsEditing(typeof bool === 'boolean' ? bool : true);
    }
  };

  const handleCancel = () => {
    setIsEditing(false);
    setIsFormDirty(false);
  };

  const handleDelete = async () => {
    if (!item.mutable) {
      addNotification({
        type: NOTIFICATION_TYPE.WARNING,
        title: FORM_ALERTS.FORBIDDEN,
        message: FORM_ALERTS.CANNOT_DELETE_RULE,
        crdType: ENTITY_TYPES.INSTRUMENTATION_RULE,
        target: id,
        hideFromHistory: true,
      });
    } else {
      await deleteInstrumentationRule(id);
    }
  };

  const handleSave = async (newTitle: string) => {
    if (validateForm({ withAlert: true, alertTitle: ACTION.UPDATE })) {
      const title = newTitle !== item.type ? newTitle : '';
      handleFormChange('ruleName', title);
      await updateInstrumentationRule(id, { ...formData, ruleName: title });
    }
  };

  return (
    <OverviewDrawer
      title={item.ruleName || (item.type as string)}
      icon={getInstrumentationRuleIcon(item.type)}
      isEdit={isEditing}
      isFormDirty={isFormDirty}
      onEdit={handleEdit}
      onSave={handleSave}
      onDelete={handleDelete}
      onCancel={handleCancel}
    >
      {isEditing && thisRule ? (
        <FormContainer>
          <RuleFormBody
            isUpdate
            rule={thisRule}
            formData={formData}
            formErrors={formErrors}
            handleFormChange={(...params) => {
              setIsFormDirty(true);
              handleFormChange(...params);
            }}
          />
        </FormContainer>
      ) : (
        <DataCard title={DATA_CARDS.RULE_DETAILS} data={cardData} />
      )}
    </OverviewDrawer>
  );
};
