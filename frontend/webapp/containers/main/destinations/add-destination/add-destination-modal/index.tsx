import React, { useEffect, useState } from 'react';
import { useQuery } from '@apollo/client';
import { DestinationTypeItem } from '@/types';
import { GET_DESTINATION_TYPE } from '@/graphql';
import { Modal, NavigationButtons } from '@/reuseable-components';
import { ConnectDestinationModalBody } from '../connect-destination-modal-body';
import { ChooseDestinationModalBody } from '../choose-destination-modal-body';

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
  const { data } = useQuery(GET_DESTINATION_TYPE);
  const [selectedItem, setSelectedItem] = useState<DestinationTypeItem>();
  const [destinationTypeList, setDestinationTypeList] = useState<
    DestinationTypeItem[]
  >([]);

  useEffect(() => {
    data && buildDestinationTypeList();
  }, [data]);

  function buildDestinationTypeList() {
    const destinationTypes = data?.destinationTypes?.categories || [];
    const destinationTypeList: DestinationTypeItem[] = destinationTypes.reduce(
      (acc: DestinationTypeItem[], category: any) => {
        const items = category.items.map((item: any) => ({
          category: category.name,
          displayName: item.displayName,
          imageUrl: item.imageUrl,
          supportedSignals: item.supportedSignals,
        }));
        return [...acc, ...items];
      },
      []
    );
    setDestinationTypeList(destinationTypeList);
  }
  function handleNextStep(item: DestinationTypeItem) {
    setSelectedItem(item);
  }

  function renderModalBody() {
    return selectedItem ? (
      <ConnectDestinationModalBody destination={selectedItem} />
    ) : (
      <ChooseDestinationModalBody
        data={destinationTypeList}
        onSelect={handleNextStep}
      />
    );
  }

  return (
    <Modal
      isOpen={isModalOpen}
      actionComponent={
        <ModalActionComponent
          onNext={() => {}}
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
