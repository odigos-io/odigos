import { Plus } from '@/assets/icons/overview';
import { Empty } from '@/assets/images';
import { KeyvalButton, KeyvalText } from '@/design.system';
import theme from '@/styles/palette';
import React from 'react';
import styled from 'styled-components';

interface EmptyListProps {
  title?: string;
  btnTitle?: string;
  btnAction?: () => void;
}

const EmptyListWrapper = styled.div`
  width: 100%;
  margin-top: 130px;
  display: flex;
  gap: 6px;
  flex-direction: column;
  justify-content: center;
  align-items: center;
`;
const BUTTON_STYLES = { gap: 10, height: 40 };
export function EmptyList({ title, btnTitle, btnAction }: EmptyListProps) {
  return (
    <EmptyListWrapper>
      <Empty />
      {title && (
        <>
          <KeyvalText size={14}>{title}</KeyvalText>
        </>
      )}
      {btnAction && (
        <KeyvalButton onClick={btnAction} style={BUTTON_STYLES}>
          <Plus />
          <KeyvalText size={16} weight={700} color={theme.text.dark_button}>
            {btnTitle}
          </KeyvalText>
        </KeyvalButton>
      )}
    </EmptyListWrapper>
  );
}
