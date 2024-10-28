import React, { useEffect } from 'react';
import { useSelector } from 'react-redux';
// import Notification from './notification';
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

export const NotificationManager: React.FC = () => {
  const notifications = useSelector((state: RootState) => state.notification.notifications);

  // temporary - until we fix the "theme" error on import from "design.system"
  useEffect(() => {
    if (notifications.length) alert(notifications[notifications.length - 1].message);
  }, [notifications.length]);

  return (
    <NotificationsWrapper>
      {/* {notifications
        .filter((notification) => notification.isNew)
        .map((notification) => (
          <Notification key={notification.id} {...notification} />
        ))} */}
    </NotificationsWrapper>
  );
};
