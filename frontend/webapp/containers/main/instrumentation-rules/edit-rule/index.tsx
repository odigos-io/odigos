'use client';
import React, { useEffect, useState } from 'react';
import theme from '@/styles/palette';
import { useSearchParams } from 'next/navigation';
import {
  INSTRUMENTATION_RULES,
  INSTRUMENTATION_RULES_DOCS_LINK,
} from '@/utils';

import {
  KeyvalButton,
  KeyvalCheckbox,
  KeyvalInput,
  KeyvalLink,
  KeyvalLoader,
  KeyvalSwitch,
  KeyvalText,
  KeyvalTextArea,
} from '@/design.system';
import {
  HeaderText,
  LoaderWrapper,
  DescriptionWrapper,
  CreateButtonWrapper,
  CreateRuleWrapper,
  KeyvalInputWrapper,
  TextareaWrapper,
  SwitchWrapper,
  FormFieldsWrapper,
} from './styled';
import { useInstrumentationRules } from '@/hooks'; // Custom hook for handling rules
import { InstrumentationRuleSpec, PayloadCollection } from '@/types';

const RULE_ID = 'id';

export function EditInstrumentationRuleContainer(): React.JSX.Element {
  const [currentRule, setCurrentRule] = useState<
    InstrumentationRuleSpec | undefined
  >();

  const search = useSearchParams();
  const { getRuleById, updateRule, toggleRuleStatus } =
    useInstrumentationRules();

  useEffect(() => {
    const ruleId = search.get(RULE_ID);
    if (!ruleId) return;

    getRule(ruleId);
  }, [search]);

  async function getRule(ruleId: string) {
    const rule = await getRuleById(ruleId);
    if (!rule) return;

    setCurrentRule(rule);
  }

  // Correctly handle state updates ensuring required fields remain intact
  function onChangeRuleState(key: keyof InstrumentationRuleSpec, value: any) {
    setCurrentRule((prev) => {
      if (!prev) return undefined;
      return { ...prev, [key]: value };
    });
  }

  // Properly handle payloadCollection updates considering it may be undefined
  function handleCheckboxChange(key: keyof PayloadCollection) {
    setCurrentRule((prev) => {
      if (!prev) return undefined;
      const payloadCollection = prev.payloadCollection || {};
      return {
        ...prev,
        payloadCollection: {
          ...payloadCollection,
          [key]: !payloadCollection[key],
        },
      };
    });
  }

  function upsertRule() {
    if (!currentRule) return;

    // Create a deep clone of the currentRule and modify the payloadCollection generically
    const modifiedRule = {
      ...currentRule,
      payloadCollection: Object.entries(
        currentRule.payloadCollection || {}
      ).reduce((acc, [key, value]) => {
        // Remove properties where the value is false; otherwise, keep the original value
        if (value !== false) {
          acc[key] = {};
        }
        return acc;
      }, {} as PayloadCollection),
    };

    updateRule({ data: modifiedRule });
  }

  if (!currentRule) return <LoaderWrapper>{<KeyvalLoader />}</LoaderWrapper>;

  return (
    <CreateRuleWrapper>
      <HeaderText>
        <KeyvalText size={18} weight={700}>
          {INSTRUMENTATION_RULES['payload-collection'].TITLE}
        </KeyvalText>
      </HeaderText>
      <DescriptionWrapper>
        <KeyvalText size={14}>
          {INSTRUMENTATION_RULES['payload-collection'].DESCRIPTION}
        </KeyvalText>
        <KeyvalLink
          value={'Documentation Link'}
          fontSize={14}
          onClick={() =>
            window.open(`${INSTRUMENTATION_RULES_DOCS_LINK}`, '_blank')
          }
        />
      </DescriptionWrapper>
      <FormFieldsWrapper disabled={false}>
        <KeyvalInputWrapper>
          <KeyvalInput
            label={'Rule Name'}
            value={currentRule?.ruleName || ''}
            onChange={(name) => onChangeRuleState('ruleName', name)}
          />
        </KeyvalInputWrapper>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <KeyvalText size={14}>
            {'Select the type of data you want to collect'}
          </KeyvalText>
          <div
            style={{
              display: 'flex',
              flexDirection: 'column',
              gap: '16px',
              marginLeft: '16px',
            }}
          >
            <KeyvalCheckbox
              label="Collect HTTP Request"
              value={!!currentRule?.payloadCollection?.httpRequest}
              onChange={() => handleCheckboxChange('httpRequest')}
            />
            <KeyvalCheckbox
              label="Collect HTTP Response"
              value={!!currentRule?.payloadCollection?.httpResponse}
              onChange={() => handleCheckboxChange('httpResponse')}
            />
            <KeyvalCheckbox
              label="Collect DB Query"
              value={!!currentRule?.payloadCollection?.dbQuery}
              onChange={() => handleCheckboxChange('dbQuery')}
            />
            <KeyvalCheckbox
              label="Collect Messaging"
              value={!!currentRule?.payloadCollection?.messaging}
              onChange={() => handleCheckboxChange('messaging')}
            />
          </div>
        </div>

        <TextareaWrapper>
          <KeyvalTextArea
            label={'Note'}
            value={currentRule?.notes || ''}
            placeholder={'Add notes here...'}
            onChange={(e) => onChangeRuleState('notes', e.target.value)}
          />
        </TextareaWrapper>
        <CreateButtonWrapper>
          <KeyvalButton onClick={upsertRule}>
            <KeyvalText weight={600} color={theme.text.dark_button} size={14}>
              {'Update Rule'}
            </KeyvalText>
          </KeyvalButton>
        </CreateButtonWrapper>
      </FormFieldsWrapper>
    </CreateRuleWrapper>
  );
}
