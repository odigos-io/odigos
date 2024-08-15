import Image from 'next/image';
import React from 'react';
import styled from 'styled-components';
import { NavigationButtons, Text } from '@/reuseable-components';

interface SetupHeaderProps {
  navigationButtons: {
    label: string;
    iconSrc?: string;
    onClick: () => void;
    variant?: 'primary' | 'secondary';
    disabled?: boolean;
  }[];
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

export const SetupHeader: React.FC<SetupHeaderProps> = ({
  navigationButtons,
}) => {
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
      <NavigationButtons buttons={navigationButtons} />
    </HeaderContainer>
  );
};
