import styled from 'styled-components';
import { ChooseRuleBody } from '../choose-rule-body';
import { RULE_OPTIONS, RuleOption } from './rule-options';
import React, { useEffect, useMemo, useState } from 'react';
import { useInstrumentationRuleCRUD } from '@/hooks/instrumentation-rules/useInstrumentationRuleCRUD';
import { useInstrumentationRuleFormData } from '@/hooks/instrumentation-rules/useInstrumentationRuleFormData';
import { AutocompleteInput, Divider, FadeLoader, Modal, NavigationButtons, NotificationNote, SectionTitle } from '@/reuseable-components';

const Container = styled.section`
  width: 100%;
  max-width: 640px;
  height: 640px;
  margin: 0 15vw;
  padding: 64px 12px 0 12px;
  display: flex;
  flex-direction: column;
  overflow-y: scroll;
`;

const Center = styled.div`
  width: 100%;
  margin-top: 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
`;

interface Props {
  isModalOpen: boolean;
  handleCloseModal: () => void;
}

export const AddRuleModal: React.FC<Props> = ({ isModalOpen, handleCloseModal }) => {
  const { formData, handleFormChange, resetFormData, validateForm } = useInstrumentationRuleFormData();
  const { createInstrumentationRule, loading } = useInstrumentationRuleCRUD({ onSuccess: handleClose });
  const [selectedItem, setSelectedItem] = useState<RuleOption | undefined>(undefined);

  useEffect(() => {
    if (!selectedItem) handleSelect(RULE_OPTIONS[0]);
  }, [selectedItem]);

  const isFormOk = useMemo(() => !!selectedItem && validateForm(), [selectedItem, formData]);

  const handleSubmit = async () => {
    createInstrumentationRule(formData);
  };

  function handleClose() {
    resetFormData();
    setSelectedItem(undefined);
    handleCloseModal();
  }

  const handleSelect = (item?: RuleOption) => {
    resetFormData();
    // handleFormChange('type', item?.type || '');
    setSelectedItem(item);
  };

  return (
    <Modal
      isOpen={isModalOpen}
      onClose={handleClose}
      header={{ title: 'Add Instrumentation Rule' }}
      actionComponent={
        <NavigationButtons
          buttons={[
            {
              variant: 'primary',
              label: 'DONE',
              onClick: handleSubmit,
              disabled: !isFormOk || loading,
            },
          ]}
        />
      }
    >
      <Container>
        <SectionTitle
          title='Define Instrumentation Rule'
          description='Instrumentation rules control how telemetry is recorded from your application. Choose a rule type and provide necessary information.'
        />
        <NotificationNote
          type='info'
          text='We currently support one rule. We’ll be adding new rule types in the near future.'
          style={{ marginTop: '24px' }}
        />
        <AutocompleteInput
          disabled
          options={RULE_OPTIONS}
          selectedOption={selectedItem}
          onOptionSelect={handleSelect}
          style={{ marginTop: '12px' }}
        />

        {!!selectedItem?.type ? (
          <div>
            <Divider margin='16px 0' />

            {loading ? (
              <Center>
                <FadeLoader cssOverride={{ scale: 2 }} />
              </Center>
            ) : (
              <ChooseRuleBody rule={selectedItem} formData={formData} handleFormChange={handleFormChange} />
            )}
          </div>
        ) : null}
      </Container>
    </Modal>
  );
};
