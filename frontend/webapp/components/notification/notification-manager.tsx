import React, { useRef, useState } from 'react';
import { useClickNotif } from '@/hooks';
import { hexPercentValues } from '@/styles';
import { useNotificationStore } from '@/store';
import { ACTION, getStatusIcon } from '@/utils';
import styled, { useTheme } from 'styled-components';
import { NotificationIcon, TrashIcon } from '@/assets';
import { useOnClickOutside, useTimeAgo } from '@/hooks';
import { NOTIFICATION_TYPE, type Notification } from '@/types';
import { IconButton, NoDataFound, Text } from '@/reuseable-components';

const RelativeContainer = styled.div`
  position: relative;
`;

const AbsoluteContainer = styled.div`
  position: absolute;
  top: 40px;
  right: 0;
  z-index: 1;
  width: 370px;
  height: 400px;
  background-color: ${({ theme }) => theme.colors.dropdown_bg};
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 24px;
  box-shadow: 0px 10px 15px -3px ${({ theme }) => theme.colors.primary}, 0px 4px 6px -2px ${({ theme }) => theme.colors.primary};
`;

const PopupHeader = styled.div`
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 24px;
  border-bottom: 1px solid ${({ theme }) => theme.colors.border};
`;

const PopupBody = styled.div`
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px;
  height: calc(100% - 74px);
  border-radius: 24px;
  overflow-y: auto;
`;

const PopupShadow = styled.div`
  position: absolute;
  bottom: 0;
  left: 0;
  width: 100%;
  height: 45px;
  border-radius: 0 0 24px 24px;
  background: ${({ theme }) => `linear-gradient(to top, ${theme.colors.dropdown_bg}, transparent)`};
  pointer-events: none;
`;

const NewCount = styled(Text)`
  background-color: ${({ theme }) => theme.colors.orange_soft};
  color: ${({ theme }) => theme.text.primary};
  border-radius: 32px;
  width: fit-content;
  padding: 2px 8px;
`;

export const NotificationManager = () => {
  const theme = useTheme();

  const { notifications: n, markAsSeen } = useNotificationStore();
  const notifications = n.filter(({ hideFromHistory }) => !hideFromHistory);
  const unseen = notifications.filter(({ seen }) => !seen);
  const unseenCount = unseen.length;

  const [isOpen, setIsOpen] = useState(false);
  const toggleOpen = () => setIsOpen((prev) => !prev);

  const containerRef = useRef<HTMLDivElement>(null);

  useOnClickOutside(containerRef, () => {
    if (isOpen) {
      setIsOpen(false);
      if (!!unseenCount) unseen.forEach(({ id }) => markAsSeen(id));
    }
  });

  return (
    <RelativeContainer ref={containerRef}>
      <IconButton data-id='notif-manager-button' onClick={toggleOpen} tooltip='Notifications' withPing={!!unseenCount} pingColor={theme.colors.orange_og}>
        <NotificationIcon size={18} />
      </IconButton>

      {isOpen && (
        <AbsoluteContainer data-id='notif-manager-content'>
          <PopupHeader>
            <Text size={20}>Notifications</Text>
            {!!unseenCount && (
              <NewCount size={12} family='secondary'>
                {unseenCount} new
              </NewCount>
            )}
          </PopupHeader>
          <PopupBody>
            {!notifications.length ? (
              <NoDataFound title='No notifications' subTitle='' />
            ) : (
              notifications.map((notif) => <NotificationListItem key={`notification-${notif.id}`} {...notif} onClick={() => setIsOpen(false)} />)
            )}
          </PopupBody>
          <PopupShadow />
        </AbsoluteContainer>
      )}
    </RelativeContainer>
  );
};

const NotifCard = styled.div`
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px;
  border-radius: 16px;
  background-color: ${({ theme }) => theme.colors.dropdown_bg_2 + hexPercentValues['080']};
  cursor: not-allowed;
  &.click-enabled {
    cursor: pointer;
    &:hover {
      background-color: ${({ theme }) => theme.colors.dropdown_bg_2};
    }
  }
`;

const StatusIcon = styled.div<{ $type: NOTIFICATION_TYPE }>`
  background-color: ${({ $type, theme }) => theme.text[$type] + hexPercentValues['015']};
  border-radius: 8px;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
`;

const NotifTextWrap = styled.div`
  width: 290px;
`;

const NotifHeaderTextWrap = styled.div`
  margin-bottom: 6px;
`;

const NotifFooterTextWrap = styled.div`
  display: flex;
  align-items: center;
  gap: 6px;
`;

const NotificationListItem: React.FC<Notification & { onClick: () => void }> = ({ onClick, ...props }) => {
  const theme = useTheme();

  const { id, seen, type, title, message, time, crdType, target } = props;
  const canClick = !!crdType && !!target;

  const isDeleted = title?.toLowerCase().includes(ACTION.DELETE.toLowerCase()) || false;
  const Icon = getStatusIcon(type);
  const timeAgo = useTimeAgo();
  const clickNotif = useClickNotif();

  return (
    <NotifCard
      key={`notification-${id}`}
      className={canClick ? 'click-enabled' : ''}
      onClick={() => {
        if (canClick) {
          onClick(); // this is to close the popup in a controlled manner, to prevent from all notifications being marked as "seen"
          clickNotif(props);
        }
      }}
    >
      <StatusIcon $type={isDeleted ? NOTIFICATION_TYPE.ERROR : type}>{isDeleted ? <TrashIcon /> : <Icon />}</StatusIcon>

      <NotifTextWrap>
        <NotifHeaderTextWrap>
          <Text size={14}>{message}</Text>
        </NotifHeaderTextWrap>

        <NotifFooterTextWrap>
          <Text size={10} color={theme.text.grey}>
            {timeAgo.format(new Date(time))}
          </Text>
          {!seen && (
            <>
              <Text size={10}>·</Text>
              <Text size={10} color={theme.colors.orange_soft}>
                new
              </Text>
            </>
          )}
        </NotifFooterTextWrap>
      </NotifTextWrap>
    </NotifCard>
  );
};
