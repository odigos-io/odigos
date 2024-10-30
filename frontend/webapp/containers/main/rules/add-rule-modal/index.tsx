import React from 'react';
import styled from 'styled-components';
import { Modal, NavigationButtons, SectionTitle } from '@/reuseable-components';

const Container = styled.section`
  width: 100%;
  max-width: 640px;
  height: 640px;
  margin: 0 15vw;
  padding-top: 64px;
  display: flex;
  flex-direction: column;
  overflow-y: scroll;
`;

interface Props {
  isModalOpen: boolean;
  handleCloseModal: () => void;
}

export const AddRuleModal: React.FC<Props> = ({ isModalOpen, handleCloseModal }) => {
  function handleClose() {
    handleCloseModal();
  }

  return (
    <Modal
      isOpen={isModalOpen}
      onClose={handleClose}
      header={{ title: 'Add Instrumentation Rule' }}
      actionComponent={
        <NavigationButtons
          buttons={[
            {
              variant: 'primary',
              label: 'DONE',
              onClick: () => {},
              disabled: true,
            },
          ]}
        />
      }
    >
      <Container>
        <SectionTitle
          title='Define Instrumentation Rule'
          description='Instrumentation rules control how telemetry is recorded from your application.'
        />
      </Container>
    </Modal>
  );
};
