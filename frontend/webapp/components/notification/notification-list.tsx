import React, { use, useEffect, useRef, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { RootState, markAsSeen } from '@/store';
import NotificationListItem from './notification-list-item';
import styled from 'styled-components';
import theme from '@/styles/palette';
import { Bell } from '@/assets/icons/app';
import { KeyvalText } from '@/design.system';
import { transform } from 'typescript';

const NotificationListContainer = styled.div`
  position: absolute;
  top: 30px;
  right: 20px;
  background-color: ${({ theme }) => theme.colors.dark};
  border: 1px solid ${theme.colors.blue_grey};
  border-radius: 8px;
  max-height: 400px;
  overflow-y: auto;
  z-index: 1000;
`;

const BellIconWrapper = styled.div`
  position: relative;
  cursor: pointer;
`;

const NotificationBadge = styled.div`
  position: absolute;
  top: -4px;
  right: -4px;
  background-color: red;
  color: white;
  border-radius: 50%;
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
`;

const NotificationList: React.FC = () => {
  const [showNotifications, setShowNotifications] = useState(false);
  const notifications = useSelector(
    (state: RootState) => state.notification.notifications
  );

  const dispatch = useDispatch();

  const isInitialRender = useRef(true);
  const unseenCount = notifications.filter(
    (notification) => !notification.seen
  ).length;
  useEffect(() => {
    if (isInitialRender.current) {
      isInitialRender.current = false;
      return;
    }

    if (!showNotifications) {
      markAllAsSeen();
    }
  }, [showNotifications]);

  function markAllAsSeen() {
    notifications.forEach((notification) => {
      if (!notification.seen) {
        dispatch(markAsSeen(notification.id));
      }
    });
  }

  return (
    <>
      <BellIconWrapper onClick={() => setShowNotifications(!showNotifications)}>
        <Bell width={24} height={24} />
        {unseenCount > 0 && (
          <NotificationBadge>
            <KeyvalText size={10}>{unseenCount}</KeyvalText>
          </NotificationBadge>
        )}
        {showNotifications && (
          <NotificationListContainer>
            {notifications.map((notification) => (
              <NotificationListItem
                key={notification.id}
                id={notification.id}
                message={notification.message}
                type={notification.type}
                seen={notification.seen}
                title={notification.title}
              />
            ))}
          </NotificationListContainer>
        )}
      </BellIconWrapper>
    </>
  );
};

export default NotificationList;
