import React from 'react';
import { SETUP } from '@/utils';
import styled from 'styled-components';
import { KeyvalText } from '@/design.system';
import { NotificationList } from '@/components';
import { BackIcon } from '@keyval-dev/design-system';
import { OdigosDescriptionDrawer } from '@/containers';

export interface OverviewHeaderProps {
  title?: string;
  onBackClick?: any;
  isDisabled?: boolean;
}

const OverviewHeaderContainer = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-right: 24px;
  border-bottom: 2px solid rgba(255, 255, 255, 0.08);
  background: ${({ theme }) => theme.colors.light_dark};
`;

const HeaderTop = styled.div`
  display: flex;
  flex-direction: column;
  margin-top: 2vh;
  margin-left: 24px;
  margin-bottom: 2vh;
  gap: 8px;
`;

const BackButtonWrapper = styled.div`
  display: flex;
  cursor: pointer;
  p {
    cursor: pointer !important;
  }
`;

export function OverviewHeader({ title, onBackClick }: OverviewHeaderProps) {
  return (
    <OverviewHeaderContainer>
      <HeaderTop>
        {onBackClick && (
          <BackButtonWrapper onClick={onBackClick}>
            <BackIcon size={14} />
            <KeyvalText size={14}>{SETUP.BACK}</KeyvalText>
          </BackButtonWrapper>
        )}
        <KeyvalText size={32} weight={700}>
          {title}
        </KeyvalText>
      </HeaderTop>
      <div style={{ display: 'flex', gap: 8 }}>
        {!onBackClick && <NotificationList />}
        {title === 'Overview' && <OdigosDescriptionDrawer />}
      </div>
    </OverviewHeaderContainer>
  );
}
