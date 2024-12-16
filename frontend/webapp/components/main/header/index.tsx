import React from 'react';
import { FlexRow } from '@/styles';
import styled from 'styled-components';
import { PlatformTypes } from '@/types';
import { OdigosLogoText } from '@/assets';
import { PlatformTitle } from './cp-title';
import { useConnectionStore } from '@/store';
import { ConnectionStatus } from '@/reuseable-components';
import { DescribeOdigos, NotificationManager, SlackInvite } from '@/components';

interface MainHeaderProps {}

const HeaderContainer = styled(FlexRow)`
  width: 100%;
  padding: 12px 0;
  background-color: ${({ theme }) => theme.colors.dark_grey};
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
  gap: 12px;
`;

export const MainHeader: React.FC<MainHeaderProps> = () => {
  const { connecting, active, title, message } = useConnectionStore();

  return (
    <HeaderContainer>
      <AlignLeft>
        <OdigosLogoText size={20} />
        <PlatformTitle type={PlatformTypes.K8S} />
        {!connecting && <ConnectionStatus title={title} subtitle={message} isActive={active} />}
      </AlignLeft>

      <AlignRight>
        <NotificationManager />
        <DescribeOdigos />
        <SlackInvite />
      </AlignRight>
    </HeaderContainer>
  );
};
