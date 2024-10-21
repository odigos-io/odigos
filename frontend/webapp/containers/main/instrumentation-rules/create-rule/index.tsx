'use client';
import React, { useEffect, useState } from 'react';
import { ActionIcon } from '@/components';
import { useRouter, useSearchParams } from 'next/navigation';
import {
  KeyvalButton,
  KeyvalCheckbox,
  KeyvalInput,
  KeyvalLink,
  KeyvalLoader,
  KeyvalText,
  KeyvalTextArea,
} from '@/design.system';
import { ACTION, ACTION_ITEM_DOCS_LINK, INSTRUMENTATION_RULES } from '@/utils';
import {
  Container,
  HeaderText,
  LoaderWrapper,
  DescriptionWrapper,
  CreateActionWrapper,
  KeyvalInputWrapper,
  TextareaWrapper,
  CreateButtonWrapper,
} from './styled';
import theme from '@/styles/palette';
import { useInstrumentationRules } from '@/hooks'; // Custom hook for instrumentation rules
import { InstrumentationRuleSpec } from '@/types';

const TYPE = 'type';

export function CreateInstrumentationRulesContainer(): React.JSX.Element {
  const [ruleType, setRuleType] = useState<string | null>(null);
  const [ruleName, setRuleName] = useState<string>('');
  const [ruleNote, setRuleNote] = useState<string>('');
  const [isFormValid, setIsFormValid] = useState<boolean>(false);
  const [payloadCollection, setPayloadCollection] = useState({
    httpRequest: false,
    httpResponse: false,
    dbQuery: false,
    messaging: false,
  });
  const search = useSearchParams();
  const router = useRouter();
  // Using the custom hook for CRUD operations
  const { addRule } = useInstrumentationRules();

  useEffect(() => {
    const type = search.get(TYPE);
    setRuleType(type);
  }, [search]);

  useEffect(() => {
    Object.values(payloadCollection).some((value) => value)
      ? setIsFormValid(true)
      : setIsFormValid(false);
  }, [payloadCollection]);

  // Function to handle checkbox change
  const handleCheckboxChange = (key: string) => {
    setPayloadCollection((prev) => ({ ...prev, [key]: !prev[key] }));
  };

  // Function to create the rule using the custom hook
  const createRule = async () => {
    // Dynamically construct the payloadCollection based on the selected checkboxes
    const constructedPayloadCollection = {
      ...(payloadCollection.httpRequest && { httpRequest: {} }),
      ...(payloadCollection.httpResponse && { httpResponse: {} }),
      ...(payloadCollection.dbQuery && { dbQuery: {} }),
      ...(payloadCollection.messaging && { messaging: {} }),
    };

    // Constructing the payload with the correct types
    const payload: InstrumentationRuleSpec = {
      ruleName,
      notes: ruleNote,
      disabled: false,
      payloadCollection: constructedPayloadCollection,
    };

    try {
      await addRule(payload); // Using the addRule function from the custom hook
      router.push('/instrumentation-rules');
    } catch (error) {
      console.error('Error creating rule:', error);
    }
  };
  if (!ruleType)
    return (
      <LoaderWrapper>
        <KeyvalLoader />
      </LoaderWrapper>
    );

  return (
    <Container>
      <CreateActionWrapper>
        <HeaderText>
          <ActionIcon type={ruleType} style={{ width: 34, height: 34 }} />
          <KeyvalText size={18} weight={700}>
            {INSTRUMENTATION_RULES[ruleType].TITLE}
          </KeyvalText>
        </HeaderText>
        <DescriptionWrapper>
          <KeyvalText size={14}>
            {INSTRUMENTATION_RULES[ruleType].DESCRIPTION}
          </KeyvalText>
          <KeyvalLink
            value={ACTION.LINK_TO_DOCS}
            fontSize={14}
            onClick={() =>
              window.open(
                `${ACTION_ITEM_DOCS_LINK}/${ruleType.toLowerCase()}`,
                '_blank'
              )
            }
          />
        </DescriptionWrapper>
        <KeyvalInputWrapper>
          <KeyvalInput
            value={ruleName}
            label="Rule Name"
            onChange={(value) => setRuleName(value)}
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
              value={payloadCollection.httpRequest}
              onChange={() => handleCheckboxChange('httpRequest')}
            />
            <KeyvalCheckbox
              label="Collect HTTP Response"
              value={payloadCollection.httpResponse}
              onChange={() => handleCheckboxChange('httpResponse')}
            />
            <KeyvalCheckbox
              label="Collect DB Query"
              value={payloadCollection.dbQuery}
              onChange={() => handleCheckboxChange('dbQuery')}
            />
            <KeyvalCheckbox
              label="Collect Messaging"
              value={payloadCollection.messaging}
              onChange={() => handleCheckboxChange('messaging')}
            />
          </div>
        </div>
        <TextareaWrapper>
          <KeyvalTextArea
            value={ruleNote}
            label="Note"
            onChange={(e) => setRuleNote(e.target.value)}
          />
        </TextareaWrapper>
        <CreateButtonWrapper>
          <KeyvalButton onClick={createRule} disabled={!isFormValid}>
            <KeyvalText size={14} weight={700} color={theme.text.dark_button}>
              Create Rule
            </KeyvalText>
          </KeyvalButton>
        </CreateButtonWrapper>
      </CreateActionWrapper>
    </Container>
  );
}
