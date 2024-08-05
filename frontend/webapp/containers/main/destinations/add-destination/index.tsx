import React, { useState } from 'react';

import styled from 'styled-components';
import { AddDestinationButton } from '@/components';
import { SectionTitle } from '@/reuseable-components';
import { AddDestinationModal } from './add-destination-modal';

const AddDestinationButtonWrapper = styled.div`
  width: 100%;
  margin-top: 24px;
`;

export function ChooseDestinationContainer() {
  const [isModalOpen, setModalOpen] = useState(false);

  const handleOpenModal = () => setModalOpen(true);
  const handleCloseModal = () => setModalOpen(false);

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
      />
    </>
  );
}
