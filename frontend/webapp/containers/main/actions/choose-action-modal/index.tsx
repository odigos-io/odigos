import { ActionInput } from '@/types';
import styled from 'styled-components';
import { useMutation } from 'react-query';
import React, { useMemo, useState } from 'react';
import { useComputePlatform, useNotify } from '@/hooks';
import { ChooseActionBody } from '../choose-action-body';
import { ACTION_OPTIONS, type ActionOption } from './action-options';
import { useActionFormData, useCreateAction } from '@/hooks/actions';
import { AutocompleteInput, Modal, NavigationButtons, Text, Divider, FadeLoader } from '@/reuseable-components';

const DefineActionContainer = styled.section`
  height: 640px;
  padding: 0px 220px;
  display: flex;
  flex-direction: column;
  overflow-y: scroll;
`;

const WidthConstraint = styled.div`
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  align-items: flex-start;
  gap: 16px;
  max-width: 640px;
  margin: 32px 0 24px 0;
`;

const SubTitle = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
  line-height: 150%;
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
  const { createNewAction } = useCreateAction();
  const { refetch } = useComputePlatform();
  const notify = useNotify();

  const [selectedItem, setSelectedItem] = useState<ActionOption | null>(null);

  const { mutate: create, isLoading } = useMutation((data: ActionInput) => createNewAction(data), {
    onSuccess: () => {
      refetch();
      handleClose();
    },
    onError: (error, variables) => {
      notify({
        message: (error as any)?.message || `Failed to create ${variables.type}: unknown error`,
        title: 'Create Error',
        type: 'error',
        target: 'notification',
        crdType: 'notification',
      });
    },
  });

  const isFormOk = useMemo(() => !!selectedItem && validateForm(), [selectedItem, formData]);

  const handleSubmit = async () => {
    if (!isFormOk) {
      notify({
        message: 'Required fields are incomplete!',
        title: 'Create Error',
        type: 'error',
        target: 'notification',
        crdType: 'notification',
      });
    } else {
      create(formData);
    }
  };

  const handleClose = () => {
    resetFormData();
    setSelectedItem(null);
    handleCloseModal();
  };

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
              disabled: !isFormOk,
            },
          ]}
        />
      }
    >
      <DefineActionContainer>
        <WidthConstraint>
          <Text size={20}>{'Define Action'}</Text>
          <SubTitle>
            {
              'Actions are a way to modify the OpenTelemetry data recorded by Odigos sources before it is exported to your Odigos destinations. Choose an action type and provide necessary information.'
            }
          </SubTitle>
        </WidthConstraint>

        <AutocompleteInput options={ACTION_OPTIONS} onOptionSelect={handleSelect} />

        {!!selectedItem?.type ? (
          <WidthConstraint>
            <Divider margin='16px 0' />

            {isLoading ? (
              <Center>
                <FadeLoader cssOverride={{ scale: 2 }} />
              </Center>
            ) : (
              <ChooseActionBody action={selectedItem} formData={formData} handleFormChange={handleFormChange} />
            )}
          </WidthConstraint>
        ) : null}
      </DefineActionContainer>
    </Modal>
  );
};
