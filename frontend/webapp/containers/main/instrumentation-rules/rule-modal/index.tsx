import React, { useState } from 'react';
import { RuleFormBody } from '../';
import { ModalBody } from '@/styles';
import { NOTIFICATION_TYPE } from '@/types';
import { ACTION, FORM_ALERTS } from '@/utils';
import { RULE_OPTIONS, RuleOption } from './rule-options';
import { CenterThis, Divider, FadeLoader, NotificationNote } from '@odigos/ui-components';
import { AutocompleteInput, Modal, NavigationButtons, SectionTitle } from '@/reuseable-components';
import { useDescribeOdigos, useInstrumentationRuleCRUD, useInstrumentationRuleFormData, useKeyDown } from '@/hooks';

interface Props {
  isOpen: boolean;
  onClose: () => void;
}

export const RuleModal: React.FC<Props> = ({ isOpen, onClose }) => {
  useKeyDown({ key: 'Enter', active: isOpen }, () => handleSubmit());

  const { isPro } = useDescribeOdigos();
  const { createInstrumentationRule, loading } = useInstrumentationRuleCRUD({ onSuccess: handleClose });
  const { formData, formErrors, handleFormChange, resetFormData, validateForm } = useInstrumentationRuleFormData();

  const [selectedItem, setSelectedItem] = useState<RuleOption | undefined>(undefined);

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
              disabled: !isPro || !selectedItem || loading,
              tooltip: !isPro ? FORM_ALERTS.ENTERPRISE_ONLY('Instrumentation Rules') : '',
            },
          ]}
        />
      }
    >
      <ModalBody>
        <SectionTitle title='Select Instrumentation Rule' description='Define how telemetry is recorded from your application. Choose a rule type and configure the details.' />
        {!isPro && <NotificationNote type={NOTIFICATION_TYPE.DEFAULT} message={FORM_ALERTS.ENTERPRISE_ONLY('Instrumentation Rules')} style={{ marginTop: '24px' }} />}
        <AutocompleteInput options={RULE_OPTIONS} selectedOption={selectedItem} onOptionSelect={handleSelect} style={{ marginTop: isPro ? '24px' : '12px' }} autoFocus={!selectedItem?.type} />

        {!!selectedItem?.type ? (
          <div>
            <Divider margin='16px 0' />

            {loading ? (
              <CenterThis>
                <FadeLoader scale={2} />
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
