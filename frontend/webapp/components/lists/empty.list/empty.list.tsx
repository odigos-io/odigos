import React from 'react';
import theme from '@/styles/palette';
import styled from 'styled-components';
import { Empty } from '@/assets/images';
import { PlusIcon } from '@keyval-dev/design-system';
import { KeyvalButton, KeyvalText } from '@/design.system';

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
        <KeyvalButton data-cy={'add-action-button'} onClick={btnAction} style={BUTTON_STYLES}>
          <PlusIcon />
          <KeyvalText size={16} weight={700} color={theme.text.dark_button}>
            {btnTitle}
          </KeyvalText>
        </KeyvalButton>
      )}
    </EmptyListWrapper>
  );
}
