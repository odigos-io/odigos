import React, { forwardRef, useImperativeHandle, useMemo } from 'react';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import { CardDetails } from '@/components';
import { ChooseRuleBody } from '../choose-rule-body';
import { RULE_OPTIONS } from '../add-rule-modal/rule-options';
import type { InstrumentationRuleInput, InstrumentationRuleSpec } from '@/types';
import { useInstrumentationRuleFormData } from '@/hooks/instrumentation-rules/useInstrumentationRuleFormData';

export type RuleDrawerHandle = {
  getCurrentData: () => InstrumentationRuleInput | null;
};

interface Props {
  isEditing: boolean;
}

const RuleDrawer = forwardRef<RuleDrawerHandle, Props>(({ isEditing }, ref) => {
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);
  const { formData, handleFormChange, resetFormData, validateForm, loadFormWithDrawerItem } = useInstrumentationRuleFormData();

  const cardData = useMemo(() => {
    if (!selectedItem) return [];

    // const { item } = selectedItem as { item: InstrumentationRuleSpec };
    // const arr = buildCardFromActionSpec(item);

    return [];
  }, [selectedItem]);

  const thisRule = useMemo(() => {
    if (!selectedItem || !isEditing) {
      resetFormData();
      return undefined;
    }

    // TODO: add support for multi rules
    const found = RULE_OPTIONS[0];

    if (!found) return undefined;

    loadFormWithDrawerItem(selectedItem);

    return found;
  }, [selectedItem, isEditing]);

  useImperativeHandle(ref, () => ({
    getCurrentData: () => (validateForm() ? formData : null),
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
