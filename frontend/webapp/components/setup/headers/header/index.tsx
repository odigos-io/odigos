import { Text } from '@/reuseable-components';
import Image from 'next/image';
import React from 'react';
import styled from 'styled-components';

interface SetupHeaderProps {
  onBack?: () => void;
  onNext?: () => void;
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

const NavigationButtons = styled.div`
  display: flex;
  align-items: center;
`;

const Button = styled.button`
  background-color: #333;
  border: none;
  color: #fff;
  padding: 5px 10px;
  margin: 0 5px;
  display: flex;
  align-items: center;
  cursor: pointer;
  font-size: 1em;

  &:hover {
    background-color: #444;
  }
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
      <Title>START WITH ODIGOS</Title>
      <NavigationButtons>
        <Button onClick={onBack}>BACK</Button>
        <Button onClick={onNext}>NEXT</Button>
      </NavigationButtons>
    </HeaderContainer>
  );
};
