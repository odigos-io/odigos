import React, { useState } from 'react';
import styled from 'styled-components';
import { KeyvalText } from '@/design.system';
import { Back } from '@/assets/icons/overview';
import { SETUP } from '@/utils/constants';
import { Bell } from '@/assets/icons/app';
import NotificationList from '@/components/notification/notification-list';

export interface OverviewHeaderProps {
  title?: string;
  onBackClick?: any;
  isDisabled?: boolean;
}

const OverviewHeaderContainer = styled.div`
  display: flex;
  /* flex-direction: column; */
  width: 100%;
  border-bottom: 2px solid rgba(255, 255, 255, 0.08);
  background: ${({ theme }) => theme.colors.light_dark};
`;

const HeaderTop = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 24px;
`;

const BackButtonWrapper = styled.div`
  display: flex;
  cursor: pointer;
  p {
    cursor: pointer !important;
  }
`;

const TextWrapper = styled.div`
  margin-top: 2vh;
  margin-left: 24px;
  margin-bottom: 2vh;
`;

const BellIconWrapper = styled.div`
  position: relative;
  cursor: pointer;
`;

export function OverviewHeader({ title, onBackClick }: OverviewHeaderProps) {
  const [showNotifications, setShowNotifications] = useState(false);

  return (
    <OverviewHeaderContainer>
      <HeaderTop>
        {onBackClick && (
          <BackButtonWrapper onClick={onBackClick}>
            <Back width={14} />
            <KeyvalText size={14}>{SETUP.BACK}</KeyvalText>
          </BackButtonWrapper>
        )}
        <TextWrapper>
          <KeyvalText size={32} weight={700}>
            {title}
          </KeyvalText>
        </TextWrapper>
      </HeaderTop>
      <BellIconWrapper onClick={() => setShowNotifications(!showNotifications)}>
        <Bell width={24} height={24} />
        {showNotifications && <NotificationList />}
      </BellIconWrapper>
    </OverviewHeaderContainer>
  );
}
