import React, { useCallback } from 'react';
import { Button, Modal, Text } from '@/reuseable-components';
import styled from 'styled-components';

interface AddActionModalProps {
  title: string;
  description: string;
  isModalOpen: boolean;
  handleDelete: () => void;
  handleCloseModal: () => void;
}

export const DeleteEntityModal: React.FC<AddActionModalProps> = ({
  isModalOpen,
  handleCloseModal,
  title = '',
  handleDelete,
  description = '',
}) => {
  const handleClose = useCallback(() => {
    handleCloseModal();
  }, [handleCloseModal]);

  return (
    <Modal isOpen={isModalOpen} onClose={handleClose}>
      <DeleteEntityModalContainer>
        <ModalTitle>Delete {title}</ModalTitle>
        <ModalContent>
          <ModalDescription>{description}</ModalDescription>
        </ModalContent>
        <ModalFooter>
          <FooterButton variant="danger" onClick={handleDelete}>
            Delete
          </FooterButton>
          <FooterButton variant={'secondary'} onClick={handleClose}>
            Cancel
          </FooterButton>
        </ModalFooter>
      </DeleteEntityModalContainer>
    </Modal>
  );
};

const ModalTitle = styled(Text)`
  font-size: 20px;
  line-height: 28px;
`;

const ModalDescription = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
  width: 416px;
  font-style: normal;
  font-weight: 300;
  line-height: 24px;
`;

const ModalContent = styled.div`
  padding: 12px 0px 32px 0;
`;

const ModalFooter = styled.div`
  display: flex;
  justify-content: space-between;
  gap: 12px;
`;

const FooterButton = styled(Button)`
  width: 224px;
`;

const DeleteEntityModalContainer = styled.div`
  padding: 24px 32px;
`;
