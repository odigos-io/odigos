import { CenterThis, ModalBody } from '@/styles';
import { ChooseRuleBody } from '../choose-rule-body';
import { RULE_OPTIONS, RuleOption } from './rule-options';
import React, { useMemo, useState } from 'react';
import { useInstrumentationRuleCRUD, useInstrumentationRuleFormData } from '@/hooks';
import { AutocompleteInput, Divider, FadeLoader, Modal, NavigationButtons, NotificationNote, SectionTitle } from '@/reuseable-components';

interface Props {
  isOpen: boolean;
  onClose: () => void;
}

export const AddRuleModal: React.FC<Props> = ({ isOpen, onClose }) => {
  const { formData, handleFormChange, resetFormData, validateForm } = useInstrumentationRuleFormData();
  const { createInstrumentationRule, loading } = useInstrumentationRuleCRUD({ onSuccess: handleClose });
  const [selectedItem, setSelectedItem] = useState<RuleOption | undefined>(RULE_OPTIONS[0]);

  const isFormOk = useMemo(() => !!selectedItem && validateForm(), [selectedItem, formData]);

  const handleSubmit = async () => {
    createInstrumentationRule(formData);
  };

  function handleClose() {
    resetFormData();
    setSelectedItem(undefined);
    onClose();
  }

  const handleSelect = (item?: RuleOption) => {
    resetFormData();
    setSelectedItem(item);
  };

  return (
    <Modal
      isOpen={isOpen}
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
      <ModalBody>
        <SectionTitle title='Define Instrumentation Rule' description='Define how telemetry is recorded from your application. Choose a rule type and configure the details.' />
        <NotificationNote type='info' message='We currently support one rule. We’ll be adding new rule types in the near future.' style={{ marginTop: '24px' }} />
        <AutocompleteInput disabled options={RULE_OPTIONS} selectedOption={selectedItem} onOptionSelect={handleSelect} style={{ marginTop: '12px' }} />

        {!!selectedItem?.type ? (
          <div>
            <Divider margin='16px 0' />

            {loading ? (
              <CenterThis>
                <FadeLoader cssOverride={{ scale: 2 }} />
              </CenterThis>
            ) : (
              <ChooseRuleBody rule={selectedItem} formData={formData} handleFormChange={handleFormChange} />
            )}
          </div>
        ) : null}
      </ModalBody>
    </Modal>
  );
};
