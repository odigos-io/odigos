import React, { forwardRef, useImperativeHandle, useMemo } from 'react';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import { CardDetails } from '@/components';
import { ChooseRuleBody } from '../choose-rule-body';
import { RULE_OPTIONS } from '../add-rule-modal/rule-options';
import buildCardFromRuleSpec from './build-card-from-rule-spec';
import { useInstrumentationRuleFormData, useNotify } from '@/hooks';
import type { InstrumentationRuleInput, InstrumentationRuleSpec } from '@/types';

export type RuleDrawerHandle = {
  getCurrentData: () => InstrumentationRuleInput | null;
};

interface Props {
  isEditing: boolean;
}

const RuleDrawer = forwardRef<RuleDrawerHandle, Props>(({ isEditing }, ref) => {
  const notify = useNotify();
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);
  const { formData, handleFormChange, resetFormData, validateForm, loadFormWithDrawerItem } = useInstrumentationRuleFormData();

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

  useImperativeHandle(ref, () => ({
    getCurrentData: () => {
      if (validateForm()) {
        return formData;
      } else {
        notify({
          message: 'Required fields are missing!',
          title: 'Update Rule Error',
          type: 'error',
          target: 'notification',
          crdType: 'notification',
        });
        return null;
      }
    },
  }));

  return isEditing && thisRule ? (
    <FormContainer>
      <ChooseRuleBody isUpdate rule={thisRule} formData={formData} handleFormChange={handleFormChange} />
    </FormContainer>
  ) : (
    <CardDetails data={cardData} />
  );
});

RuleDrawer.displayName = 'RuleDrawer';

export { RuleDrawer };

const FormContainer = styled.div`
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  padding-right: 16px;
  box-sizing: border-box;
  overflow: overlay;
  max-height: calc(100vh - 220px);
`;
