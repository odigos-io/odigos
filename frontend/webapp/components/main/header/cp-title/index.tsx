import React from 'react';
import { K8sLogo } from '@/assets';
import styled from 'styled-components';
import { PlatformTypes } from '@/types';
import { Text } from '@/reuseable-components';

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
  color: ${({ theme }) => theme.colors.white};
`;

export const PlatformTitle: React.FC<Props> = ({ type }) => {
  if (PlatformTypes.K8S) {
    return (
      <Container>
        <K8sLogo size={28} />
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
