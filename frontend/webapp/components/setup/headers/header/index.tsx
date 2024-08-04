import Image from 'next/image';
import React from 'react';
import styled from 'styled-components';
import { NavigationButtons, Text } from '@/reuseable-components';

interface SetupHeaderProps {
  onBack: () => void;
  onNext: () => void;
}

const HeaderContainer = styled.div`
  display: flex;
  justify-content: space-between;
  padding: 0 24px 0 32px;
  align-items: center;
  background-color: ${({ theme }) => theme.colors.dark_grey};
  border-bottom: 1px solid rgba(249, 249, 249, 0.16);
  height: 80px;
`;

const Title = styled(Text)``;

const Logo = styled.div`
  display: flex;
  align-items: center;
  font-size: 1.2em;
`;

export const SetupHeader: React.FC<SetupHeaderProps> = ({ onBack, onNext }) => {
  return (
    <HeaderContainer>
      <Logo>
        <Image
          src="/brand/transparent-logo-white.svg"
          alt="logo"
          width={84}
          height={20}
        />
      </Logo>
      <Title family={'secondary'}>START WITH ODIGOS</Title>
      <NavigationButtons
        buttons={[
          {
            label: 'BACK',
            iconSrc: '/icons/common/arrow-white.svg',
            onClick: onBack,
            variant: 'secondary',
          },
          {
            label: 'NEXT',
            iconSrc: '/icons/common/arrow-black.svg',
            onClick: onNext,
            variant: 'primary',
          },
        ]}
      />
    </HeaderContainer>
  );
};
