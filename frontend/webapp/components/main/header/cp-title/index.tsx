import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { PlatformTypes } from '@/types';
import { Text } from '@/reuseable-components';

interface Props {
  type: PlatformTypes;
}

const PlatformWrapper = styled.div`
  display: flex;
  align-items: center;
  padding: 10px;
`;

const IconWrapper = styled.div`
  margin-right: 10px;
`;

const TextWrapper = styled.div`
  display: flex;
  align-items: center;
`;

const Title = styled(Text)`
  font-size: 14px;
  margin-right: 10px;
  color: ${({ theme }) => theme.colors.white};
`;

const PlatformTitle: React.FC<Props> = ({ type }) => {
  return (
    <PlatformWrapper>
      <IconWrapper>
        <Image src={`/icons/cp/${type}.svg`} alt={type} width={28} height={28} />
      </IconWrapper>
      <TextWrapper>
        <Title>{type === PlatformTypes.K8S ? 'Kubernetes Cluster' : 'Virtual Machine'}</Title>
      </TextWrapper>
    </PlatformWrapper>
  );
};

export { PlatformTitle };
