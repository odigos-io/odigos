import { ChooseActionBody } from '../';
import React, { useMemo, useState } from 'react';
import { CenterThis, ModalBody } from '@/styles';
import { useActionCRUD, useActionFormData } from '@/hooks/actions';
import { ACTION_OPTIONS, type ActionOption } from './action-options';
import { AutocompleteInput, Modal, NavigationButtons, Divider, FadeLoader, SectionTitle } from '@/reuseable-components';

interface AddActionModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const AddActionModal: React.FC<AddActionModalProps> = ({ isOpen, onClose }) => {
  const { formData, handleFormChange, resetFormData, validateForm } = useActionFormData();
  const { createAction, loading } = useActionCRUD({ onSuccess: handleClose });
  const [selectedItem, setSelectedItem] = useState<ActionOption | undefined>(undefined);

  const isFormOk = useMemo(() => !!selectedItem && validateForm(), [selectedItem, formData]);

  const handleSubmit = async () => {
    createAction(formData);
  };

  function handleClose() {
    resetFormData();
    setSelectedItem(undefined);
    onClose();
  }

  const handleSelect = (item?: ActionOption) => {
    resetFormData();
    handleFormChange('type', item?.type || '');
    setSelectedItem(item);
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      header={{ title: 'Add Action' }}
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
        <SectionTitle
          title='Define Action'
          description='Actions are a way to modify the OpenTelemetry data recorded by Odigos sources before it is exported to your Odigos destinations. Choose an action type and provide necessary information.'
        />
        <AutocompleteInput options={ACTION_OPTIONS} selectedOption={selectedItem} onOptionSelect={handleSelect} style={{ marginTop: '24px' }} />

        {!!selectedItem?.type ? (
          <div>
            <Divider margin='16px 0' />

            {loading ? (
              <CenterThis>
                <FadeLoader cssOverride={{ scale: 2 }} />
              </CenterThis>
            ) : (
              <ChooseActionBody action={selectedItem} formData={formData} handleFormChange={handleFormChange} />
            )}
          </div>
        ) : null}
      </ModalBody>
    </Modal>
  );
};
