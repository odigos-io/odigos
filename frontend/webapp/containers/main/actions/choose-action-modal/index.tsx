import styled from 'styled-components';
import { ChooseActionBody } from '../';
import React, { useMemo, useState } from 'react';
import { useActionCRUD, useActionFormData } from '@/hooks/actions';
import { ACTION_OPTIONS, type ActionOption } from './action-options';
import { AutocompleteInput, Modal, NavigationButtons, Text, Divider, FadeLoader, SectionTitle } from '@/reuseable-components';

const Container = styled.section`
  width: 100%;
  max-width: 640px;
  height: 640px;
  margin: 0 15vw;
  padding-top: 64px;
  display: flex;
  flex-direction: column;
  overflow-y: scroll;
`;

const Center = styled.div`
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
`;

interface AddActionModalProps {
  isModalOpen: boolean;
  handleCloseModal: () => void;
}

export const AddActionModal: React.FC<AddActionModalProps> = ({ isModalOpen, handleCloseModal }) => {
  const { formData, handleFormChange, resetFormData, validateForm } = useActionFormData();
  const { createAction, loading } = useActionCRUD({ onSuccess: handleClose });
  const [selectedItem, setSelectedItem] = useState<ActionOption | null>(null);

  const isFormOk = useMemo(() => !!selectedItem && validateForm(), [selectedItem, formData]);

  const handleSubmit = async () => {
    createAction(formData);
  };

  function handleClose() {
    resetFormData();
    setSelectedItem(null);
    handleCloseModal();
  }

  const handleSelect = (item: ActionOption) => {
    resetFormData();
    handleFormChange('type', item.type);
    setSelectedItem(item);
  };

  return (
    <Modal
      isOpen={isModalOpen}
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
      <Container>
        <SectionTitle
          title='Define Action'
          description='Actions are a way to modify the OpenTelemetry data recorded by Odigos sources before it is exported to your Odigos destinations. Choose an action type and provide necessary information.'
        />
        <AutocompleteInput options={ACTION_OPTIONS} onOptionSelect={handleSelect} />

        {!!selectedItem?.type ? (
          <div>
            <Divider margin='16px 0' />

            {loading ? (
              <Center>
                <FadeLoader cssOverride={{ scale: 2 }} />
              </Center>
            ) : (
              <ChooseActionBody action={selectedItem} formData={formData} handleFormChange={handleFormChange} />
            )}
          </div>
        ) : null}
      </Container>
    </Modal>
  );
};
