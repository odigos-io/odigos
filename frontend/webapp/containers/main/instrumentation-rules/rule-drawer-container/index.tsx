import React, { useMemo, useState } from 'react';
import styled from 'styled-components';
import { getRuleIcon } from '@/utils';
import { useDrawerStore } from '@/store';
import { CardDetails } from '@/components';
import { ChooseRuleBody } from '../choose-rule-body';
import type { InstrumentationRuleSpec } from '@/types';
import OverviewDrawer from '../../overview/overview-drawer';
import { RULE_OPTIONS } from '../add-rule-modal/rule-options';
import buildCardFromRuleSpec from './build-card-from-rule-spec';
import { useInstrumentationRuleCRUD, useInstrumentationRuleFormData } from '@/hooks';

interface Props {}

const RuleDrawer: React.FC<Props> = () => {
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);
  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);

  const { formData, handleFormChange, resetFormData, validateForm, loadFormWithDrawerItem } = useInstrumentationRuleFormData();
  const { updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD();

  const cardData = useMemo(() => {
    if (!selectedItem) return [];

    const { item } = selectedItem as { item: InstrumentationRuleSpec };
    const arr = buildCardFromRuleSpec(item);

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
    await deleteInstrumentationRule(id as string);
  };

  const handleSave = async () => {
    if (validateForm({ withAlert: true })) {
      await updateInstrumentationRule(id as string, formData);
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
          <ChooseRuleBody
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

export { RuleDrawer };

const FormContainer = styled.div`
  width: 100%;
  height: 100%;
  max-height: calc(100vh - 220px);
  overflow: overlay;
  overflow-y: auto;
`;
