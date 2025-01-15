import React from 'react';
import { K8sLogo } from '@/assets';
import { PlatformTypes } from '@/types';
import { Text } from '@/reuseable-components';
import styled, { useTheme } from 'styled-components';
import { useDarkModeStore } from '@/store';

interface Props {
  type: PlatformTypes;
}

const Container = styled.div`
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
`;

const Title = styled(Text)`
  font-size: 14px;
  margin-right: 10px;
  color: ${({ theme }) => theme.text.secondary};
`;

const LogoWrap = styled.div<{ $darkMode: boolean }>`
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px;
  border-radius: 100%;
  background-color: ${({ theme, $darkMode }) => theme[$darkMode ? 'colors' : 'text'].info};
`;

export const PlatformTitle: React.FC<Props> = ({ type }) => {
  const theme = useTheme();
  const { darkMode } = useDarkModeStore();

  if (type === PlatformTypes.K8S) {
    return (
      <Container>
        <LogoWrap $darkMode={darkMode}>
          <K8sLogo size={20} fill={theme[darkMode ? 'text' : 'colors'].info} />
        </LogoWrap>
        <Title>Kubernetes Cluster</Title>
      </Container>
    );
  }

  return (
    <Container>
      <Title>Virtual Machine</Title>
    </Container>
  );
};
