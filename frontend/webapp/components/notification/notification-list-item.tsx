import React from 'react';
import theme from '@/styles/palette';
import styled from 'styled-components';
import { ROUTES, timeAgo } from '@/utils';
import { useRouter } from 'next/navigation';
import { getIcon } from './notification-icon';
import { KeyvalLink, KeyvalText } from '@/design.system';
import { NotificationButtonContainer, NotificationDetailsWrapper } from './notification-container';
import { Notification } from '@/types';

const NotificationItemContainer = styled.div<{ seen: boolean }>`
  border-bottom: 1px solid ${theme.colors.blue_grey};
  padding: 10px;
  gap: 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: ${({ seen }) => (seen ? theme.colors.light_dark : theme.colors.dark)};

  &:hover {
    background-color: ${theme.colors.dark};
  }
`;

const NotificationContent = styled.div`
  display: flex;
  width: 300px;
  gap: 8px;
`;

const NotificationListItem: React.FC<Notification> = ({ message, type, seen, title, crdType, target, time }) => {
  const router = useRouter();

  function onDetailsClick() {
    if (target) {
      switch (crdType) {
        case 'Destination':
          router.push(`${ROUTES.UPDATE_DESTINATION}${target}`);
          break;
        case 'InstrumentedApplication':
        case 'InstrumentationInstance':
          router.push(`${ROUTES.MANAGE_SOURCE}?${target}`);
          break;
        default:
          break;
      }
    }
  }
  return (
    <NotificationItemContainer seen={seen}>
      <NotificationContent>
        <div>{getIcon(type)}</div>
        <NotificationDetailsWrapper>
          <KeyvalText size={16} weight={600}>
            {title}
          </KeyvalText>
          <KeyvalText size={14}>{message}</KeyvalText>
          {time && (
            <KeyvalText color={theme.text.light_grey} size={12}>
              {timeAgo(time)}
            </KeyvalText>
          )}
        </NotificationDetailsWrapper>
      </NotificationContent>
      <NotificationButtonContainer>{!!target && <KeyvalLink fontSize={12} value='Details' onClick={onDetailsClick} />}</NotificationButtonContainer>
    </NotificationItemContainer>
  );
};

export default NotificationListItem;
