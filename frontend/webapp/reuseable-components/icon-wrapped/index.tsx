import React, { useState } from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { OdigosLogo, type SVG } from '@/assets';

interface Props {
  icon?: SVG;
  src?: string;
  alt?: string;
  isError?: boolean;
}

const Container = styled.div<{ $isError: Props['isError'] }>`
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: ${({ $isError }) =>
    $isError ? 'linear-gradient(180deg, rgba(237, 124, 124, 0.2) 0%, rgba(237, 124, 124, 0.05) 100%);' : 'linear-gradient(180deg, rgba(249, 249, 249, 0.2) 0%, rgba(249, 249, 249, 0.05) 100%);'};
`;

export const IconWrapped: React.FC<Props> = ({ icon: Icon, src = '', alt = '', isError }) => {
  const [srcHasError, setSrcHasError] = useState(false);

  if (!!src && !srcHasError) {
    return (
      <Container $isError={isError}>
        <Image src={src} alt={alt} width={20} height={20} onError={() => setSrcHasError(true)} />
      </Container>
    );
  }

  return <Container $isError={isError}>{!!Icon ? <Icon size={20} /> : <OdigosLogo size={16} />}</Container>;
};
