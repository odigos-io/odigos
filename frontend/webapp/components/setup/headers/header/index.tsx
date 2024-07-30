import { Button, Text } from '@/reuseable-components';
import theme from '@/styles/theme';
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

const HeaderButton = styled(Button)`
  display: flex;
  align-items: center;
  gap: 8px;
`;

const Logo = styled.div`
  display: flex;
  align-items: center;
  font-size: 1.2em;
`;

const NavigationButtons = styled.div`
  display: flex;
  gap: 8px;
  align-items: center;
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
      <NavigationButtons>
        <HeaderButton variant="secondary" onClick={onBack}>
          <Image
            src="/icons/common/arrow-white.svg"
            alt="back"
            width={8}
            height={12}
          />
          <Text
            color={theme.colors.secondary}
            size={14}
            decoration={'underline'}
            family="secondary"
          >
            BACK
          </Text>
        </HeaderButton>
        <HeaderButton variant="primary" onClick={onNext}>
          <Text
            decoration={'underline'}
            color={theme.colors.dark_grey}
            size={14}
            family="secondary"
          >
            NEXT
          </Text>
          <Image
            src="/icons/common/arrow-black.svg"
            alt="next"
            width={8}
            height={12}
          />
        </HeaderButton>
      </NavigationButtons>
    </HeaderContainer>
  );
};
