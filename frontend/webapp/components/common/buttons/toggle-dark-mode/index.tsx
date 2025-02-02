import React, { useEffect } from 'react';
import styled from 'styled-components';
import { useDarkModeStore } from '@/store';
import { FlexRow, MoonIcon, SunIcon } from '@odigos/ui-components';

interface Props {}

const Container = styled(FlexRow)`
  position: relative;

  padding: 6px;
  gap: 6px;
  border-radius: 32px;
  border: 1px solid ${({ theme }) => theme.colors.border};

  & > svg {
    cursor: pointer;
    z-index: 1;
  }
  &:hover {
    border: 1px solid ${({ theme }) => theme.colors.secondary};
  }
`;

const Background = styled.div<{ $darkMode: boolean }>`
  position: absolute;
  top: 2px;
  left: ${({ $darkMode }) => ($darkMode ? '2px' : 'calc(100% - 2px - 24px)')};
  z-index: 0;
  width: 24px;
  height: 24px;
  background-color: ${({ theme }) => theme.colors.border};
  border-radius: 100%;
  transition: all 0.3s;
`;

export const ToggleDarkMode: React.FC<Props> = () => {
  const { darkMode, setDarkMode } = useDarkModeStore();

  useEffect(() => {
    const lsValue = localStorage.getItem('darkMode');
    if (!!lsValue) setDarkMode(lsValue == 'true');
  }, []);

  const handleToggle = (bool?: boolean) => {
    const val = typeof bool === 'boolean' ? bool : !darkMode;
    setDarkMode(val);
    localStorage.setItem('darkMode', JSON.stringify(val));
  };

  return (
    <Container onClick={() => handleToggle()}>
      <MoonIcon />
      <SunIcon />
      <Background $darkMode={darkMode} />
    </Container>
  );
};
