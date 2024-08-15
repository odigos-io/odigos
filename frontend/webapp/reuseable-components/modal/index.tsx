import React from 'react';
import Image from 'next/image';
import { Text } from '../text';
import ReactDOM from 'react-dom';
import styled from 'styled-components';
import { Button } from '../button';

interface ModalProps {
  isOpen: boolean;
  header: {
    title: string;
  };
  actionComponent?: React.ReactNode;
  onClose: () => void;
  children: React.ReactNode;
}

const Overlay = styled.div`
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(17, 17, 17, 0.8);
  backdrop-filter: blur(1px);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
`;

const ModalWrapper = styled.div`
  background: ${({ theme }) => theme.colors.translucent_bg};
  border-radius: 40px;
  border: ${({ theme }) => `1px solid ${theme.colors.border}`};
  box-shadow: 0px 1px 1px 0px rgba(17, 17, 17, 0.8),
    0px 2px 2px 0px rgba(17, 17, 17, 0.8), 0px 5px 5px 0px rgba(17, 17, 17, 0.8),
    0px 10px 10px 0px rgba(17, 17, 17, 0.8),
    0px 0px 8px 0px rgba(17, 17, 17, 0.8);
`;

const ModalHeader = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 80px;
  border-bottom: 1px solid ${({ theme }) => theme.colors.border};
  padding: 0 24px;
`;

const ModalCloseButton = styled.div`
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
`;

const HeaderActionsWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;

const ModalContent = styled.div``;

const ModalTitleContainer = styled.div`
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  pointer-events: none;
`;

const ModalTitle = styled(Text)`
  text-transform: uppercase;
  font-family: ${({ theme }) => theme.font_family.secondary};
  pointer-events: auto;
`;

const CancelText = styled(Text)`
  text-transform: uppercase;
  font-weight: 500;
  font-size: 14px;
  font-family: ${({ theme }) => theme.font_family.secondary};
  text-decoration: underline;
  cursor: pointer;
`;

const Modal: React.FC<ModalProps> = ({
  isOpen,
  header,
  onClose,
  children,
  actionComponent,
}) => {
  if (!isOpen) return null;

  return ReactDOM.createPortal(
    <Overlay>
      <ModalWrapper>
        <ModalHeader>
          <ModalCloseButton onClick={onClose}>
            <Image
              src="/icons/common/x.svg"
              alt="close"
              width={15}
              height={12}
            />
            <CancelText>{'Cancel'}</CancelText>
          </ModalCloseButton>
          <ModalTitleContainer>
            <ModalTitle>{header.title}</ModalTitle>
          </ModalTitleContainer>
          <HeaderActionsWrapper>{actionComponent}</HeaderActionsWrapper>
        </ModalHeader>
        <ModalContent>{children}</ModalContent>
      </ModalWrapper>
    </Overlay>,
    document.body
  );
};

export { Modal };
