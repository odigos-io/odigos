// DrawerFooter.tsx
import React, { useEffect, useState } from 'react';
import Image from 'next/image';
import styled, { css } from 'styled-components';
import { Button, Text } from '@/reuseable-components';

interface Props {
  onSave: () => void;
  saveLabel?: string;
  onCancel: () => void;
  cancelLabel?: string;
  onDelete: () => void;
  deleteLabel?: string;
}

const FooterContainer = styled.div<{ $isVisible: boolean }>`
  display: flex;
  justify-content: space-between;
  gap: 8px;

  padding: 24px 18px 24px 32px;
  border-top: 1px solid ${({ theme }) => theme.colors.border};
  background-color: ${({ theme }) => theme.colors.translucent_bg};

  opacity: 0;
  transform: translateY(20px);
  transition: opacity 0.3s ease, transform 0.3s ease;

  ${({ $isVisible }) =>
    $isVisible &&
    css`
      opacity: 1;
      transform: translateY(0);
    `}
`;

const AlignRight = styled.div`
  margin-left: auto;
`;

const FooterButton = styled(Button)`
  width: 140px;
`;

const ButtonText = styled(Text)<{ $variant?: 'primary' | 'secondary' | 'tertiary' }>`
  color: ${({ theme, $variant }) => ($variant === 'primary' ? theme.text.primary : $variant === 'tertiary' ? theme.text.error : theme.text.secondary)};
  font-size: 14px;
  font-weight: 600;
  font-family: ${({ theme }) => theme.font_family.secondary};
  text-transform: uppercase;
  width: fit-content;
`;

const DrawerFooter: React.FC<Props> = ({ onSave, saveLabel = 'Save', onCancel, cancelLabel = 'Cancel', onDelete, deleteLabel = 'Delete' }) => {
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    // Trigger animation on mount
    setIsVisible(true);
  }, []);

  return (
    <FooterContainer $isVisible={isVisible}>
      <FooterButton variant='primary' onClick={onSave}>
        <ButtonText $variant='primary'>{saveLabel}</ButtonText>
      </FooterButton>
      <FooterButton variant='secondary' onClick={onCancel}>
        <ButtonText>{cancelLabel}</ButtonText>
      </FooterButton>

      <AlignRight>
        <FooterButton variant='tertiary' onClick={onDelete}>
          <Image src='/icons/common/trash.svg' alt='Delete' width={16} height={16} />
          <ButtonText $variant='tertiary'>{deleteLabel}</ButtonText>
        </FooterButton>
      </AlignRight>
    </FooterContainer>
  );
};

export default DrawerFooter;
