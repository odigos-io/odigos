import React, { useEffect, useState } from 'react';

import styled from 'styled-components';
import { useQuery } from '@apollo/client';
import { DestinationTypeItem } from '@/types';
import { GET_DESTINATION_TYPE } from '@/graphql';
import { AddDestinationButton } from '@/components';
import { SectionTitle } from '@/reuseable-components';
import { AddDestinationModal } from './add-destination-modal';

const AddDestinationButtonWrapper = styled.div`
  width: 100%;
  margin-top: 24px;
`;

export function ChooseDestinationContainer() {
  const { data } = useQuery(GET_DESTINATION_TYPE);

  const [isModalOpen, setModalOpen] = useState(false);
  const [destinationTypeList, setDestinationTypeList] = useState<
    DestinationTypeItem[]
  >([]);

  const handleOpenModal = () => setModalOpen(true);
  const handleCloseModal = () => setModalOpen(false);

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

  return (
    <>
      <SectionTitle
        title="Configure destinations"
        description="Add backend destinations where collected data will be sent and configure their settings."
      />
      <AddDestinationButtonWrapper>
        <AddDestinationButton onClick={() => handleOpenModal()} />
      </AddDestinationButtonWrapper>
      <AddDestinationModal
        isModalOpen={isModalOpen}
        handleCloseModal={handleCloseModal}
        destinationTypeList={destinationTypeList}
      />
    </>
  );
}
