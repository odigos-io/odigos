import React, { useState, useRef } from 'react';
import { DestinationTypeItem } from '@/types';
import { Modal, NavigationButtons } from '@/reuseable-components';
import { ChooseDestinationModalBody } from '../choose-destination-modal-body';
import { ConnectDestinationModalBody } from '../connect-destination-modal-body';

interface AddDestinationModalProps {
  isModalOpen: boolean;
  handleCloseModal: () => void;
}

function ModalActionComponent({
  onNext,
  onBack,
  item,
}: {
  onNext: () => void;
  onBack: () => void;
  item: DestinationTypeItem | undefined;
}) {
  return (
    <NavigationButtons
      buttons={
        item
          ? [
              {
                label: 'BACK',
                iconSrc: '/icons/common/arrow-white.svg',
                onClick: onBack,
                variant: 'secondary',
              },
              {
                label: 'DONE',
                onClick: onNext,
                variant: 'primary',
              },
            ]
          : []
      }
    />
  );
}

export function AddDestinationModal({
  isModalOpen,
  handleCloseModal,
}: AddDestinationModalProps) {
  const submitRef = useRef<() => void | null>(null);
  const [selectedItem, setSelectedItem] = useState<DestinationTypeItem>();

  function handleNextStep(item: DestinationTypeItem) {
    setSelectedItem(item);
  }

  function renderModalBody() {
    return selectedItem ? (
      <ConnectDestinationModalBody
        destination={selectedItem}
        onSubmitRef={submitRef}
      />
    ) : (
      <ChooseDestinationModalBody onSelect={handleNextStep} />
    );
  }

  function handleNext() {
    if (submitRef.current) {
      submitRef.current();
      handleCloseModal();
    }
  }

  return (
    <Modal
      isOpen={isModalOpen}
      actionComponent={
        <ModalActionComponent
          onNext={handleNext}
          onBack={() => setSelectedItem(undefined)}
          item={selectedItem}
        />
      }
      header={{ title: 'Add destination' }}
      onClose={handleCloseModal}
    >
      {renderModalBody()}
    </Modal>
  );
}
