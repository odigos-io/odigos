import React, { useState } from 'react';
import styled from 'styled-components';
import { Button, ButtonProps, Modal, NotificationNote, Text, Transition } from '@/reuseable-components';
import { useKeyDown } from '@/hooks';
import { slide } from '@/styles';

interface ButtonParams {
  text: string;
  variant?: ButtonProps['variant'];
  onClick: () => void;
}

interface Props {
  isOpen: boolean;
  noOverlay?: boolean;
  title: string;
  description: string;
  warnAgain?: {
    title: string;
    description: string;
  };
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

const NoteWrapper = styled.div`
  margin-top: 12px;
`;

export const WarningModal: React.FC<Props> = ({ isOpen, noOverlay, title = '', description = '', warnAgain, approveButton, denyButton }) => {
  useKeyDown({ key: 'Enter', active: isOpen }, () => approveButton.onClick());

  const [showWarnAgain, setShowWarnAgain] = useState(false);

  const onApprove = () => {
    warnAgain && !showWarnAgain ? setShowWarnAgain(true) : approveButton.onClick();
  };

  const onDeny = () => {
    setShowWarnAgain(false);
    denyButton.onClick();
  };

  return (
    <Modal isOpen={isOpen} noOverlay={noOverlay} onClose={onDeny}>
      <Container>
        <Title>{title}</Title>

        <Content>
          <Description>{description}</Description>
        </Content>

        <Footer>
          <FooterButton variant={approveButton.variant || 'primary'} onClick={onApprove}>
            {approveButton.text}
          </FooterButton>
          <FooterButton variant={denyButton.variant || 'secondary'} onClick={onDeny}>
            {denyButton.text}
          </FooterButton>
        </Footer>

        {!!warnAgain && (
          <Transition container={NoteWrapper} enter={showWarnAgain} animateIn={slide.in['left']} animateOut={slide.out['left']}>
            <NotificationNote type='error' title={warnAgain.title} message={warnAgain.description} />
          </Transition>
        )}
      </Container>
    </Modal>
  );
};
