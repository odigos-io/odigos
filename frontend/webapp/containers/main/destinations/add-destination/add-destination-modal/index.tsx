import React, { useState, useRef, useCallback } from 'react';
import { DestinationTypeItem } from '@/types';
import { Modal, NavigationButtons } from '@/reuseable-components';
import { ChooseDestinationModalBody } from '../choose-destination-modal-body';
import { ConnectDestinationModalBody } from '../connect-destination-modal-body';

interface AddDestinationModalProps {
  isModalOpen: boolean;
  handleCloseModal: () => void;
}

interface ModalActionComponentProps {
  onNext: () => void;
  onBack: () => void;
  isFormValid?: boolean;
  item?: DestinationTypeItem;
}

const ModalActionComponent: React.FC<ModalActionComponentProps> = React.memo(
  ({ onNext, onBack, isFormValid, item }) => {
    if (!item) return null;

    const buttons = [
      {
        label: 'BACK',
        iconSrc: '/icons/common/arrow-white.svg',
        onClick: onBack,
        variant: 'secondary' as const,
      },
      {
        label: 'DONE',
        onClick: onNext,
        variant: 'primary' as const,
        disabled: !isFormValid,
      },
    ];

    return <NavigationButtons buttons={buttons} />;
  }
);

export const AddDestinationModal: React.FC<AddDestinationModalProps> = ({
  isModalOpen,
  handleCloseModal,
}) => {
  const submitRef = useRef<(() => void) | null>(null);
  const [selectedItem, setSelectedItem] = useState<
    DestinationTypeItem | undefined
  >();
  const [isFormValid, setIsFormValid] = useState<boolean>(false);

  const handleNextStep = useCallback((item: DestinationTypeItem) => {
    setSelectedItem(item);
  }, []);

  const handleNext = useCallback(() => {
    if (submitRef.current) {
      submitRef.current();
      setSelectedItem(undefined);
      handleCloseModal();
    }
  }, [handleCloseModal]);

  const handleBack = useCallback(() => {
    setSelectedItem(undefined);
  }, []);

  const handleClose = useCallback(() => {
    setSelectedItem(undefined);
    handleCloseModal();
  }, [handleCloseModal]);

  const renderModalBody = () => {
    return selectedItem ? (
      <ConnectDestinationModalBody
        onSubmitRef={submitRef}
        destination={selectedItem}
        onFormValidChange={setIsFormValid}
      />
    ) : (
      <ChooseDestinationModalBody onSelect={handleNextStep} />
    );
  };

  return (
    <Modal
      isOpen={isModalOpen}
      actionComponent={
        <ModalActionComponent
          onNext={handleNext}
          onBack={handleBack}
          isFormValid={isFormValid}
          item={selectedItem}
        />
      }
      header={{ title: 'Add Destination' }}
      onClose={handleClose}
    >
      {renderModalBody()}
    </Modal>
  );
};
