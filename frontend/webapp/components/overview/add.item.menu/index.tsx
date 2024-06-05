import React from 'react';
import theme from '@/styles/palette';
import styled from 'styled-components';
import { PlusIcon } from '@keyval-dev/design-system';
import { KeyvalText, KeyvalButton } from '@/design.system';

const MenuWrapper = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 36px;
`;

const BUTTON_STYLES = { gap: 10, height: 40 };

interface AddItemMenuProps {
  length: number;
  onClick: () => void;
  btnLabel: string;
  lengthLabel: string;
}

export function AddItemMenu({
  length = 0,
  onClick,
  btnLabel,
  lengthLabel,
}: AddItemMenuProps) {
  return (
    <MenuWrapper>
      <KeyvalText>{`${length} ${lengthLabel}`}</KeyvalText>
      <KeyvalButton onClick={onClick} style={BUTTON_STYLES}>
        <PlusIcon />
        <KeyvalText size={16} weight={700} color={theme.text.dark_button}>
          {btnLabel}
        </KeyvalText>
      </KeyvalButton>
    </MenuWrapper>
  );
}
