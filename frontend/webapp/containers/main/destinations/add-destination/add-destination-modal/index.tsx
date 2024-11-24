import React, { useState } from 'react';
import { ModalBody } from '@/styles';
import { useAppStore } from '@/store';
import { INPUT_TYPES } from '@/utils';
import { SideMenu } from '@/components';
import { Container, SideMenuWrapper } from '../styled';
import { useDestinationCRUD, useDestinationFormData } from '@/hooks';
import type { ConfiguredDestination, DestinationTypeItem } from '@/types';
import { ChooseDestinationModalBody } from '../choose-destination-modal-body';
import { ConnectDestinationModalBody } from '../connect-destination-modal-body';
import { Modal, type NavigationButtonProps, NavigationButtons } from '@/reuseable-components';

interface AddDestinationModalProps {
  isOnboarding?: boolean;
  isOpen: boolean;
  onClose: () => void;
}

export const AddDestinationModal: React.FC<AddDestinationModalProps> = ({ isOnboarding, isOpen, onClose }) => {
  const [selectedItem, setSelectedItem] = useState<DestinationTypeItem | undefined>();

  const { createDestination } = useDestinationCRUD();
  const addConfiguredDestination = useAppStore(({ addConfiguredDestination }) => addConfiguredDestination);
  const { formData, handleFormChange, resetFormData, validateForm, dynamicFields, setDynamicFields } = useDestinationFormData({
    supportedSignals: selectedItem?.supportedSignals,
    preLoadedFields: selectedItem?.fields,
  });

  const isFormOk = !!selectedItem && validateForm();

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
      createDestination(formData);
    }

    handleClose();
  };

  const renderHeaderButtons = () => {
    const buttons: NavigationButtonProps[] = [
      {
        label: 'DONE',
        variant: 'primary' as const,
        onClick: handleSubmit,
        disabled: !isFormOk,
      },
    ];

    if (!!selectedItem) {
      buttons.unshift({
        label: 'BACK',
        iconSrc: '/icons/common/arrow-white.svg',
        variant: 'secondary' as const,
        onClick: handleBack,
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

        <ModalBody style={{ margin: '32px 24px 0 24px' }}>
          {!!selectedItem ? (
            <ConnectDestinationModalBody destination={selectedItem} formData={formData} handleFormChange={handleFormChange} dynamicFields={dynamicFields} setDynamicFields={setDynamicFields} />
          ) : (
            <ChooseDestinationModalBody onSelect={handleSelect} />
          )}
        </ModalBody>
      </Container>
    </Modal>
  );
};
