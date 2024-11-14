import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { PlatformTitle } from './cp-title';
import { Status } from '@/reuseable-components';
import { NotificationManager } from '@/components/notification';

interface MainHeaderProps {}

const Flex = styled.div`
  display: flex;
  align-items: center;
`;

const HeaderContainer = styled(Flex)`
  width: 100%;
  padding: 12px 0;
  background-color: ${({ theme }) => theme.colors.dark_grey};
  border-bottom: 1px solid rgba(249, 249, 249, 0.16);
`;

const AlignLeft = styled(Flex)`
  margin-right: auto;
  margin-left: 32px;
  gap: 16px;
`;

const AlignRight = styled(Flex)`
  margin-left: auto;
  margin-right: 32px;
  gap: 16px;
`;

export const MainHeader: React.FC<MainHeaderProps> = () => {
  return (
    <HeaderContainer>
      <AlignLeft>
        <Image src='/brand/transparent-logo-white.svg' alt='logo' width={84} height={20} />
        <PlatformTitle type='k8s' />
        <Status title='Connection Alive' isActive withIcon withBackground />
      </AlignLeft>

      <AlignRight>
        <NotificationManager />
        {/* <Flex style={{ gap: '8px' }}>
          <Image src='/icons/common/avatar.svg' alt='avatar' width={28} height={28} />
          <Text>Ben Elferink</Text>
        </Flex> */}
      </AlignRight>
    </HeaderContainer>
  );
};
