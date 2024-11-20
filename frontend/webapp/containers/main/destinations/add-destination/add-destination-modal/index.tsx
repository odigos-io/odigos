import React, { useState, useRef, useCallback } from 'react';
import type { DestinationTypeItem } from '@/types';
import { ChooseDestinationModalBody } from '../choose-destination-modal-body';
import { ConnectDestinationModalBody } from '../connect-destination-modal-body';
import { Modal, type NavigationButtonProps, NavigationButtons } from '@/reuseable-components';

interface AddDestinationModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const AddDestinationModal: React.FC<AddDestinationModalProps> = ({ isOpen, onClose }) => {
  const submitRef = useRef<(() => void) | null>(null);
  const [selectedItem, setSelectedItem] = useState<DestinationTypeItem | undefined>();
  const [isFormValid, setIsFormValid] = useState<boolean>(false);

  const handleNextStep = useCallback((item: DestinationTypeItem) => {
    setSelectedItem(item);
  }, []);

  const handleNext = useCallback(() => {
    if (submitRef.current) {
      submitRef.current();
      setSelectedItem(undefined);
      onClose();
    }
  }, [onClose]);

  const handleBack = useCallback(() => {
    setSelectedItem(undefined);
  }, []);

  const handleClose = useCallback(() => {
    setSelectedItem(undefined);
    onClose();
  }, [onClose]);

  const renderHeaderButtons = () => {
    const buttons: NavigationButtonProps[] = [
      {
        label: 'DONE',
        variant: 'primary' as const,
        disabled: !isFormValid,
        onClick: handleNext,
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

  const renderModalBody = () => {
    return selectedItem ? (
      <ConnectDestinationModalBody onSubmitRef={submitRef} destination={selectedItem} onFormValidChange={setIsFormValid} />
    ) : (
      <ChooseDestinationModalBody onSelect={handleNextStep} />
    );
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} header={{ title: 'Add Destination' }} actionComponent={<NavigationButtons buttons={renderHeaderButtons()} />}>
      {renderModalBody()}
    </Modal>
  );
};
