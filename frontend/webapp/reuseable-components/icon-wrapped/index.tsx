import React, { useState } from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { hexPercentValues } from '@/styles';
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
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: ${({ $isError, theme }) => {
    const clr = $isError ? theme.text.error : theme.text.secondary;
    return `linear-gradient(180deg, ${clr + hexPercentValues['020']} 0%, ${clr + hexPercentValues['005']} 100%)`;
  }};
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

  return <Container $isError={isError}>{!!Icon ? <Icon size={20} /> : <OdigosLogo />}</Container>;
};
