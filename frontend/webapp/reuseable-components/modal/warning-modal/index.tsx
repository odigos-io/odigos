import React from 'react';
import { Button, ButtonProps, Modal, Text } from '@/reuseable-components';
import styled from 'styled-components';

interface ButtonParams {
  text: string;
  variant?: ButtonProps['variant'];
  onClick: () => void;
}

interface Props {
  isOpen: boolean;
  title: string;
  description: string;
  approveButton: ButtonParams;
  denyButton: ButtonParams;
}

const Container = styled.div`
  padding: 24px 32px;
`;

const Content = styled.div`
  padding: 12px 0px 32px 0;
`;

const Title = styled(Text)`
  font-size: 20px;
  line-height: 28px;
`;

const Description = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
  width: 416px;
  font-style: normal;
  font-weight: 300;
  line-height: 24px;
`;

const Footer = styled.div`
  display: flex;
  justify-content: space-between;
  gap: 12px;
`;

const FooterButton = styled(Button)`
  width: 224px;
`;

export const WarningModal: React.FC<Props> = ({ isOpen, title = '', description = '', approveButton, denyButton }) => {
  return (
    <Modal isOpen={isOpen} onClose={denyButton.onClick}>
      <Container>
        <Title>{title}</Title>

        <Content>
          <Description>{description}</Description>
        </Content>

        <Footer>
          <FooterButton variant={approveButton.variant || 'primary'} onClick={approveButton.onClick}>
            {approveButton.text}
          </FooterButton>
          <FooterButton variant={denyButton.variant || 'secondary'} onClick={denyButton.onClick}>
            {denyButton.text}
          </FooterButton>
        </Footer>
      </Container>
    </Modal>
  );
};
