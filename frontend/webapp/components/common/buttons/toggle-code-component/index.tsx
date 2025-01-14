import React from 'react';
import { FlexRow } from '@/styles';
import styled from 'styled-components';
import { CodeIcon, ListIcon } from '@/assets';

interface Props {
  isCodeMode: boolean;
  setIsCodeMode: (isCodeMode: boolean) => void;
}

const Container = styled(FlexRow)`
  gap: 0;
`;

const Button = styled.button<{ $position: 'left' | 'right'; $selected: boolean }>`
  padding: 4px 8px;
  background-color: ${({ theme, $selected }) => ($selected ? theme.colors.white_opacity['008'] : 'transparent')};
  border-radius: ${({ $position }) => ($position === 'left' ? '32px 0px 0px 32px' : $position === 'right' ? '0px 32px 32px 0px' : '0')};
  border: 1px solid ${({ theme }) => theme.colors.border};
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  &:hover {
    border: 1px solid ${({ theme }) => theme.colors.secondary};
  }
`;

export const ToggleCodeComponent: React.FC<Props> = ({ isCodeMode, setIsCodeMode }) => {
  return (
    <Container>
      <Button $position='left' $selected={!isCodeMode} onClick={() => setIsCodeMode(false)}>
        <ListIcon />
      </Button>
      <Button $position='right' $selected={isCodeMode} onClick={() => setIsCodeMode(true)}>
        <CodeIcon />
      </Button>
    </Container>
  );
};
