import React, { useMemo, useState } from 'react';
import buildCard from './build-card';
import { RuleFormBody } from '../';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import { DataCard } from '@/reuseable-components';
import buildDrawerItem from './build-drawer-item';
import { RULE_OPTIONS } from '../rule-modal/rule-options';
import OverviewDrawer from '../../overview/overview-drawer';
import { ACTION, DATA_CARDS, FORM_ALERTS, getRuleIcon, NOTIFICATION } from '@/utils';
import { useInstrumentationRuleCRUD, useInstrumentationRuleFormData, useNotify } from '@/hooks';
import { InstrumentationRuleType, OVERVIEW_ENTITY_TYPES, type InstrumentationRuleSpec } from '@/types';

interface Props {}

const FormContainer = styled.div`
  width: 100%;
  height: 100%;
  max-height: calc(100vh - 220px);
  overflow: overlay;
  overflow-y: auto;
`;

export const RuleDrawer: React.FC<Props> = () => {
  const notify = useNotify();
  const { selectedItem, setSelectedItem } = useDrawerStore();
  const { formData, formErrors, handleFormChange, resetFormData, validateForm, loadFormWithDrawerItem } = useInstrumentationRuleFormData();

  const { updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD({
    onSuccess: (type) => {
      setIsEditing(false);
      setIsFormDirty(false);

      if (type === ACTION.DELETE) {
        setSelectedItem(null);
      } else {
        const { item } = selectedItem as { item: InstrumentationRuleSpec };
        const { ruleId: id } = item;
        setSelectedItem({ id, type: OVERVIEW_ENTITY_TYPES.RULE, item: buildDrawerItem(id, formData, item) });
      }
    },
  });

  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);

  const cardData = useMemo(() => {
    if (!selectedItem) return [];

    const { item } = selectedItem as { item: InstrumentationRuleSpec };
    const arr = buildCard(item);

    return arr;
  }, [selectedItem]);

  const thisRule = useMemo(() => {
    if (!selectedItem || !isEditing) {
      resetFormData();
      return undefined;
    }

    const { item } = selectedItem as { item: InstrumentationRuleSpec };
    const found = RULE_OPTIONS.find(({ type }) => type === item.type);

    if (!found) return undefined;

    loadFormWithDrawerItem(selectedItem);

    return found;
  }, [selectedItem, isEditing]);

  if (!selectedItem?.item) return null;
  const { id, item } = selectedItem as { id: string; item: InstrumentationRuleSpec };

  const handleEdit = (bool?: boolean) => {
    if (item.type === InstrumentationRuleType.UNKNOWN_TYPE) {
      notify({ type: NOTIFICATION.WARNING, title: FORM_ALERTS.FORBIDDEN, message: FORM_ALERTS.CANNOT_EDIT_RULE, crdType: OVERVIEW_ENTITY_TYPES.RULE, target: id });
    } else {
      setIsEditing(typeof bool === 'boolean' ? bool : true);
    }
  };

  const handleCancel = () => {
    setIsEditing(false);
    setIsFormDirty(false);
  };

  const handleDelete = async () => {
    await deleteInstrumentationRule(id);
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
      imageUri={getRuleIcon(item.type)}
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
