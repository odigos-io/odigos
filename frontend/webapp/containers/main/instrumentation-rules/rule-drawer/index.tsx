import React, { useMemo, useState } from 'react';
import buildCard from './build-card';
import { RuleFormBody } from '../';
import styled from 'styled-components';
import { ACTION, getRuleIcon } from '@/utils';
import { useDrawerStore } from '@/store';
import { CardDetails } from '@/components';
import { RULE_OPTIONS } from '../rule-modal/rule-options';
import OverviewDrawer from '../../overview/overview-drawer';
import { OVERVIEW_ENTITY_TYPES, type InstrumentationRuleSpec } from '@/types';
import { useInstrumentationRuleCRUD, useInstrumentationRuleFormData } from '@/hooks';
import buildDrawerItem from './build-drawer-item';

interface Props {}

const FormContainer = styled.div`
  width: 100%;
  height: 100%;
  max-height: calc(100vh - 220px);
  overflow: overlay;
  overflow-y: auto;
`;

export const RuleDrawer: React.FC<Props> = () => {
  const { selectedItem, setSelectedItem } = useDrawerStore();
  const { formData, handleFormChange, resetFormData, validateForm, loadFormWithDrawerItem } = useInstrumentationRuleFormData();

  const { updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD({
    onSuccess: (type) => {
      setIsEditing(false);
      setIsFormDirty(false);

      if (type === ACTION.DELETE) {
        setSelectedItem(null);
      } else {
        const id = (selectedItem?.item as InstrumentationRuleSpec)?.ruleId;
        setSelectedItem({ id, type: OVERVIEW_ENTITY_TYPES.RULE, item: buildDrawerItem(id, formData) });
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
  const { id, item } = selectedItem;

  const handleEdit = (bool?: boolean) => {
    setIsEditing(typeof bool === 'boolean' ? bool : true);
  };

  const handleCancel = () => {
    setIsEditing(false);
    setIsFormDirty(false);
  };

  const handleDelete = async () => {
    await deleteInstrumentationRule(id as string);
  };

  const handleSave = async (newTitle: string) => {
    if (validateForm({ withAlert: true })) {
      const title = newTitle !== ((item as InstrumentationRuleSpec).type as string) ? newTitle : '';
      await updateInstrumentationRule(id as string, { ...formData, ruleName: title });
    }
  };

  return (
    <OverviewDrawer
      title={(item as InstrumentationRuleSpec).ruleName || ((item as InstrumentationRuleSpec).type as string)}
      imageUri={getRuleIcon((item as InstrumentationRuleSpec).type)}
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
