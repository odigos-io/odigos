import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { ConnectionStatus } from '@/reuseable-components';
import { PlatformTitle } from './cp-title';

interface MainHeaderProps {}

const HeaderContainer = styled.div`
  display: flex;
  padding: 10px 0;
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

const PlatformTitleWrapper = styled.div`
  margin-left: 32px;
`;

export const MainHeader: React.FC<MainHeaderProps> = () => {
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
      <PlatformTitleWrapper>
        <PlatformTitle type="k8s" />
      </PlatformTitleWrapper>
      <ConnectionStatus
        title="Connection Status"
        status="lost"
        subtitle="Please check your internet connection"
      />
    </HeaderContainer>
  );
};
