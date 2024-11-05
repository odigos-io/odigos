import React, { useState, useRef, useCallback } from 'react';
import { DestinationTypeItem } from '@/types';
import { Modal, NavigationButtons } from '@/reuseable-components';
import { ChooseDestinationModalBody } from '../choose-destination-modal-body';
import { ConnectDestinationModalBody } from '../connect-destination-modal-body';

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

  const renderModalBody = () => {
    return selectedItem ? (
      <ConnectDestinationModalBody onSubmitRef={submitRef} destination={selectedItem} onFormValidChange={setIsFormValid} />
    ) : (
      <ChooseDestinationModalBody onSelect={handleNextStep} />
    );
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      header={{ title: 'Add Destination' }}
      actionComponent={
        <NavigationButtons
          buttons={[
            {
              label: 'BACK',
              iconSrc: '/icons/common/arrow-white.svg',
              onClick: handleBack,
              variant: 'secondary' as const,
            },
            {
              label: 'DONE',
              onClick: handleNext,
              variant: 'primary' as const,
              disabled: !isFormValid,
            },
          ]}
        />
      }
    >
      {renderModalBody()}
    </Modal>
  );
};
