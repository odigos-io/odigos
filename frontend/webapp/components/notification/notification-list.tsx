import React, { useEffect, useRef, useState } from 'react';
import styled from 'styled-components';
import theme from '@/styles/palette';
import { BellIcon } from '@keyval-dev/design-system';
import { KeyvalText } from '@/design.system';
import { useOnClickOutside } from '@/hooks';
import NotificationListItem from './notification-list-item';
import { useNotificationStore } from '@/store';

const NotificationListContainer = styled.div`
  position: absolute;
  top: 50px;
  right: 60px;
  background-color: ${theme.colors.light_dark};
  border: 1px solid ${theme.colors.blue_grey};
  border-radius: 8px;
  max-height: 400px;
  overflow-y: auto;
  z-index: 1000;
`;

const BellIconWrapper = styled.div`
  position: relative;
  padding: 6px;
  border-radius: 8px;
  border: 1px solid ${theme.colors.blue_grey};
  display: flex;
  align-items: center;
  &:hover {
    background-color: ${theme.colors.dark};
  }
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

const NotificationHeader = styled.div`
  padding: 14px;
  border-bottom: 1px solid ${theme.colors.blue_grey};
  display: flex;
  justify-content: space-between;
  align-items: center;
`;

export const NotificationList: React.FC = () => {
  const [showNotifications, setShowNotifications] = useState(false);
  const { notifications, markAsSeen } = useNotificationStore();

  const containerRef = useRef(null);
  const isInitialRender = useRef(true);
  useOnClickOutside(containerRef, () => setShowNotifications(false));
  const unseenCount = notifications.filter((notification) => !notification.seen).length;

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
        markAsSeen(notification.id);
      }
    });
  }

  return notifications.length > 0 ? (
    <div ref={containerRef}>
      <BellIconWrapper>
        <BellIcon style={{ cursor: 'pointer' }} size={20} onClick={() => setShowNotifications(!showNotifications)} />
        {unseenCount > 0 && (
          <NotificationBadge>
            <KeyvalText size={10}>{unseenCount}</KeyvalText>
          </NotificationBadge>
        )}
      </BellIconWrapper>
      {showNotifications && (
        <NotificationListContainer>
          <NotificationHeader>
            <KeyvalText size={18} weight={600}>
              Notifications
            </KeyvalText>
          </NotificationHeader>

          {[...notifications].reverse().map((notification) => (
            <NotificationListItem key={notification.id} {...notification} />
          ))}
        </NotificationListContainer>
      )}
    </div>
  ) : null;
};
