import React, { useState } from 'react';
import { ModalBody } from '@/styles';
import { useAppStore } from '@/store';
import styled from 'styled-components';
import { SideMenu } from '@/components';
import { ACTION, INPUT_TYPES } from '@/utils';
import { DestinationFormBody } from '../destination-form-body';
import { ChooseDestinationBody } from './choose-destination-body';
import { useDestinationCRUD, useDestinationFormData } from '@/hooks';
import type { ConfiguredDestination, DestinationTypeItem } from '@/types';
import { Modal, type NavigationButtonProps, NavigationButtons } from '@/reuseable-components';

interface AddDestinationModalProps {
  isOnboarding?: boolean;
  isOpen: boolean;
  onClose: () => void;
}

const Container = styled.div`
  display: flex;
`;

const SideMenuWrapper = styled.div`
  border-right: 1px solid ${({ theme }) => theme.colors.border};
  padding: 32px;
  width: 200px;
  @media (max-width: 1050px) {
    display: none;
  }
`;

export const DestinationModal: React.FC<AddDestinationModalProps> = ({ isOnboarding, isOpen, onClose }) => {
  const [selectedItem, setSelectedItem] = useState<DestinationTypeItem | undefined>();

  const { createDestination, loading } = useDestinationCRUD();
  const addConfiguredDestination = useAppStore(({ addConfiguredDestination }) => addConfiguredDestination);
  const { formData, formErrors, handleFormChange, resetFormData, validateForm, dynamicFields, setDynamicFields } = useDestinationFormData({
    supportedSignals: selectedItem?.supportedSignals,
    preLoadedFields: selectedItem?.fields,
  });

  const handleClose = () => {
    resetFormData();
    setSelectedItem(undefined);
    onClose();
  };

  const handleBack = () => {
    resetFormData();
    setSelectedItem(undefined);
  };

  const handleSelect = (item: DestinationTypeItem) => {
    resetFormData();
    handleFormChange('type', item.type);
    setSelectedItem(item);
  };

  const handleSubmit = async () => {
    const isFormOk = validateForm({ withAlert: !isOnboarding, alertTitle: ACTION.CREATE });
    if (!isFormOk) return null;

    if (isOnboarding) {
      const destinationTypeDetails = dynamicFields.map((field) => ({
        title: field.title,
        value: field.componentType === INPUT_TYPES.DROPDOWN ? field.value.value : field.value,
      }));

      destinationTypeDetails.unshift({
        title: 'Destination name',
        value: formData.name,
      });

      const storedDestination: ConfiguredDestination = {
        type: selectedItem?.type || '',
        displayName: selectedItem?.displayName || '',
        imageUrl: selectedItem?.imageUrl || '',
        exportedSignals: formData.exportedSignals,
        destinationTypeDetails,
        category: '', // Could be handled in a more dynamic way if needed
      };

      addConfiguredDestination({ stored: storedDestination, form: formData });
    } else {
      await createDestination(formData);
    }

    handleClose();
  };

  const renderHeaderButtons = () => {
    const buttons: NavigationButtonProps[] = [
      {
        label: 'DONE',
        variant: 'primary' as const,
        onClick: handleSubmit,
        disabled: !selectedItem || loading,
      },
    ];

    if (!!selectedItem) {
      buttons.unshift({
        label: 'BACK',
        iconSrc: '/icons/common/arrow-white.svg',
        variant: 'secondary' as const,
        onClick: handleBack,
        disabled: loading,
      });
    }

    return buttons;
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} header={{ title: 'Add Destination' }} actionComponent={<NavigationButtons buttons={renderHeaderButtons()} />}>
      <Container>
        <SideMenuWrapper>
          <SideMenu
            currentStep={!!selectedItem ? 2 : 1}
            data={[
              { stepNumber: 1, title: 'DESTINATIONS', state: 'active' },
              { stepNumber: 2, title: 'CONNECTION', state: 'disabled' },
            ]}
          />
        </SideMenuWrapper>

        <ModalBody style={{ margin: '32px 24px 12px 24px' }}>
          {/*
            in other modals we would render this out, but for this case we will use "hidden" instead,
            this is to preserve the filters-state when going back-and-forth between selections
          */}
          <ChooseDestinationBody onSelect={handleSelect} hidden={!!selectedItem} />

          {!!selectedItem && (
            <DestinationFormBody
              destination={selectedItem}
              formErrors={formErrors}
              formData={formData}
              handleFormChange={handleFormChange}
              dynamicFields={dynamicFields}
              setDynamicFields={setDynamicFields}
            />
          )}
        </ModalBody>
      </Container>
    </Modal>
  );
};
