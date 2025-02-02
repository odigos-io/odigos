import React from 'react';
import { PlatformTypes } from '@/types';
import { Text } from '@/reuseable-components';
import { K8sLogo } from '@odigos/ui-components';
import styled, { useTheme } from 'styled-components';

interface Props {
  type: PlatformTypes;
}

const Container = styled.div`
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px;
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 32px;
`;

const Title = styled(Text)`
  font-size: 14px;
  margin-right: 10px;
  color: ${({ theme }) => theme.text.secondary};
`;

const LogoWrap = styled.div`
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px;
  border-radius: 100%;
  background-color: ${({ theme }) => theme.colors.info};
`;

export const PlatformTitle: React.FC<Props> = ({ type }) => {
  const theme = useTheme();

  if (type === PlatformTypes.K8S) {
    return (
      <Container>
        <LogoWrap>
          <K8sLogo size={20} fill={theme.text.info} />
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
