import React from 'react';
import Image from 'next/image';
import { FlexRow } from '@/styles';
import styled from 'styled-components';
import { PlatformTypes } from '@/types';
import { PlatformTitle } from './cp-title';
import { useConnectionStore } from '@/store';
import { ConnectionStatus } from '@/reuseable-components';
import { NotificationManager } from '@/components/notification';

interface MainHeaderProps {}

const HeaderContainer = styled(FlexRow)`
  width: 100%;
  padding: 12px 0;
  background-color: ${({ theme }) => theme.colors.darker_grey};
  border-bottom: 1px solid rgba(249, 249, 249, 0.16);
`;

const AlignLeft = styled(FlexRow)`
  margin-right: auto;
  margin-left: 32px;
  gap: 16px;
`;

const AlignRight = styled(FlexRow)`
  margin-left: auto;
  margin-right: 32px;
  gap: 16px;
`;

export const MainHeader: React.FC<MainHeaderProps> = () => {
  const { connecting, active, title, message } = useConnectionStore();

  return (
    <HeaderContainer>
      <AlignLeft>
        <Image src='/brand/transparent-logo-white.svg' alt='logo' width={84} height={20} />
        <PlatformTitle type={PlatformTypes.K8S} />
        {!connecting && <ConnectionStatus title={title} subtitle={message} isActive={active} />}
      </AlignLeft>

      <AlignRight>
        <NotificationManager />
      </AlignRight>
    </HeaderContainer>
  );
};
