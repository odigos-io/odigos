import React, { useState } from 'react';
import { ACTION } from '@/utils';
import { RuleFormBody } from '../';
import { CenterThis, ModalBody } from '@/styles';
import { RULE_OPTIONS, RuleOption } from './rule-options';
import { useInstrumentationRuleCRUD, useInstrumentationRuleFormData } from '@/hooks';
import { AutocompleteInput, Divider, FadeLoader, Modal, NavigationButtons, NotificationNote, SectionTitle } from '@/reuseable-components';

interface Props {
  isOpen: boolean;
  onClose: () => void;
}

export const RuleModal: React.FC<Props> = ({ isOpen, onClose }) => {
  const { formData, formErrors, handleFormChange, resetFormData, validateForm } = useInstrumentationRuleFormData();
  const { createInstrumentationRule, loading } = useInstrumentationRuleCRUD({ onSuccess: handleClose });
  const [selectedItem, setSelectedItem] = useState<RuleOption | undefined>(RULE_OPTIONS[0]);

  function handleClose() {
    resetFormData();
    setSelectedItem(undefined);
    onClose();
  }

  const handleSelect = (item?: RuleOption) => {
    resetFormData();
    setSelectedItem(item);
  };

  const handleSubmit = async () => {
    const isFormOk = validateForm({ withAlert: true, alertTitle: ACTION.CREATE });
    if (!isFormOk) return null;

    await createInstrumentationRule(formData);

    handleClose();
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
              disabled: !selectedItem || loading,
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
              <RuleFormBody rule={selectedItem} formData={formData} formErrors={formErrors} handleFormChange={handleFormChange} />
            )}
          </div>
        ) : null}
      </ModalBody>
    </Modal>
  );
};
