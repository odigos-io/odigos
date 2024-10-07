// DrawerFooter.tsx
import React from 'react';
import styled from 'styled-components';
import { Button, Text } from '@/reuseable-components'; // Adjust the path if needed
import Image from 'next/image';

interface DrawerFooterProps {
  onSave: () => void;
  onCancel: () => void;
  onDelete: () => void;
}

const DrawerFooter: React.FC<DrawerFooterProps> = ({
  onSave,
  onCancel,
  onDelete,
}) => (
  <FooterContainer>
    <LeftButtonsWrapper>
      <FooterButton disabled variant="primary" onClick={onSave}>
        <ButtonText variant="primary">Save</ButtonText>
      </FooterButton>
      <FooterButton variant="secondary" onClick={onCancel}>
        <ButtonText>Cancel</ButtonText>
      </FooterButton>
    </LeftButtonsWrapper>
    <FooterButton variant="tertiary" onClick={onDelete}>
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

export default DrawerFooter;

const FooterContainer = styled.div`
  display: flex;
  justify-content: space-between;
  padding: 16px;
  background-color: ${({ theme }) => theme.colors.translucent_bg};
  border-top: 1px solid rgba(249, 249, 249, 0.24);
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
