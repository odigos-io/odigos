import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { ConnectionStatus } from '@/reuseable-components';

interface SetupHeaderProps {}

const HeaderContainer = styled.div`
  display: flex;
  padding: 20px 24px 20px 32px;
  align-items: center;
  background-color: ${({ theme }) => theme.colors.dark_grey};
  border-bottom: 1px solid rgba(249, 249, 249, 0.16);
  width: 100%;
`;

const Logo = styled.div`
  display: flex;
  align-items: center;
  margin-left: 32px;
`;

export const MainHeader: React.FC<SetupHeaderProps> = () => {
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
      <ConnectionStatus
        title="Connection Status"
        status="lost"
        subtitle="Please check your internet connection"
      />
    </HeaderContainer>
  );
};
