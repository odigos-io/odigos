import React from 'react';
import { useSelector } from 'react-redux';

import Notification from './notification';
import styled from 'styled-components';
import { RootState } from '@/store';

const NotificationsWrapper = styled.div`
  position: fixed;
  top: 20px;
  right: 20px;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  z-index: 1000;
`;

const NotificationManager: React.FC = () => {
  const notifications = useSelector(
    (state: RootState) => state.notification.notifications
  );

  return (
    <NotificationsWrapper>
      {notifications
        .filter((notification) => notification.isNew)
        .map((notification) => (
          <Notification
            key={notification.id}
            id={notification.id}
            message={notification.message}
            title={notification.title}
            type={notification.type}
          />
        ))}
    </NotificationsWrapper>
  );
};

export default NotificationManager;
