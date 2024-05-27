import React from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '@/store';
import NotificationListItem from './notification-list-item';
import styled from 'styled-components';
import theme from '@/styles/palette';

const NotificationListContainer = styled.div`
  position: absolute;
  top: 50px;
  right: 20px;
  background-color: ${({ theme }) => theme.colors.light_dark};
  border: 1px solid ${theme.colors.blue_grey};
  border-radius: 8px;
  width: 300px;
  max-height: 400px;
  overflow-y: auto;
  z-index: 1000;
`;

const NotificationList: React.FC = () => {
  const notifications = useSelector(
    (state: RootState) => state.notification.notifications
  );

  return (
    <NotificationListContainer>
      {notifications.map((notification) => (
        <NotificationListItem
          key={notification.id}
          id={notification.id}
          message={notification.message}
          type={notification.type}
          seen={notification.seen}
        />
      ))}
    </NotificationListContainer>
  );
};

export default NotificationList;
