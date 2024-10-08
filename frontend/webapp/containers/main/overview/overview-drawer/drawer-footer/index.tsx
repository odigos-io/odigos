// DrawerFooter.tsx
import React, { useEffect, useState } from 'react';
import Image from 'next/image';
import styled, { css } from 'styled-components';
import { Button, Text } from '@/reuseable-components';

interface DrawerFooterProps {
  onSave: () => void;
  onCancel: () => void;
  onDelete: () => void;
}

const DrawerFooter: React.FC<DrawerFooterProps> = ({
  onSave,
  onCancel,
  onDelete,
}) => {
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    // Trigger animation on mount
    setIsVisible(true);
  }, []);

  return (
    <FooterContainer isVisible={isVisible}>
      <LeftButtonsWrapper>
        <FooterButton variant="primary" onClick={onSave}>
          <ButtonText variant="primary">Save</ButtonText>
        </FooterButton>
        <FooterButton variant="secondary" onClick={onCancel}>
          <ButtonText>Cancel</ButtonText>
        </FooterButton>
      </LeftButtonsWrapper>
      <FooterButton
        style={{ width: 100 }}
        variant="tertiary"
        onClick={onDelete}
      >
        <Image
          src="/icons/common/trash.svg"
          alt="Delete"
          width={16}
          height={16}
        />
        <ButtonText variant="tertiary">Delete</ButtonText>
      </FooterButton>
    </FooterContainer>
  );
};

export default DrawerFooter;

const FooterContainer = styled.div<{ isVisible: boolean }>`
  display: flex;
  justify-content: space-between;
  padding: 24px 18px 24px 32px;
  background-color: ${({ theme }) => theme.colors.translucent_bg};
  border-top: 1px solid rgba(249, 249, 249, 0.24);
  opacity: 0;
  transform: translateY(20px);
  transition: opacity 0.3s ease, transform 0.3s ease;

  ${({ isVisible }) =>
    isVisible &&
    css`
      opacity: 1;
      transform: translateY(0);
    `}
`;

const LeftButtonsWrapper = styled.div`
  display: flex;
  gap: 8px;
`;

const FooterButton = styled(Button)`
  width: 140px;
  gap: 8px;
`;

const ButtonText = styled(Text)<{
  variant?: 'primary' | 'secondary' | 'tertiary';
}>`
  color: ${({ theme, variant }) =>
    variant === 'primary'
      ? theme.text.primary
      : variant === 'tertiary'
      ? theme.text.error
      : theme.text.secondary};
  font-size: 14px;
  font-weight: 600;
  font-family: ${({ theme }) => theme.font_family.secondary};
  text-decoration-line: underline;
  text-transform: uppercase;
  width: fit-content;
`;
