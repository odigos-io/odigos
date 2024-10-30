import React, { useEffect, useState } from 'react';
import styled from 'styled-components';
import { AutocompleteInput, Divider, FadeLoader, Modal, NavigationButtons, SectionTitle } from '@/reuseable-components';
import { RULE_OPTIONS, RuleOption } from './rule-options';
import Disclaimer from '@/reuseable-components/disclaimer';

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
  // const { formData, handleFormChange, resetFormData, validateForm } = useRuleFormData();
  // const { createRule, loading } = useRuleCRUD({ onSuccess: handleClose });
  const [selectedItem, setSelectedItem] = useState<RuleOption | undefined>(undefined);

  useEffect(() => {
    if (!selectedItem) handleSelect(RULE_OPTIONS[0]);
  }, [selectedItem]);

  // const isFormOk = useMemo(() => !!selectedItem && validateForm(), [selectedItem, formData]);

  const handleSubmit = async () => {
    // createRule(formData);
  };

  function handleClose() {
    // resetFormData();
    setSelectedItem(undefined);
    handleCloseModal();
  }

  const handleSelect = (item: RuleOption) => {
    // resetFormData();
    // handleFormChange('type', item.type);
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
              disabled: true,
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
        <Disclaimer text='We currently support one rule. We’ll be adding new rule types in the near future.' style={{ marginTop: '24px' }} />
        <AutocompleteInput options={RULE_OPTIONS} selectedOption={selectedItem} onOptionSelect={handleSelect} style={{ marginTop: '12px' }} />

        {!!selectedItem?.type ? (
          <div>
            <Divider margin='16px 0' />

            {/* {loading ? ( */}
            <Center>
              <FadeLoader cssOverride={{ scale: 2 }} />
            </Center>
            {/* ) : (
              <ChooseRuleBody rule={selectedItem} formData={formData} handleFormChange={handleFormChange} />
            )} */}
          </div>
        ) : null}
      </Container>
    </Modal>
  );
};
