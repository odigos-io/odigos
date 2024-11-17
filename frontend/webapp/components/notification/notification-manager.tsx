import React, { useEffect, useMemo, useRef, useState } from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { useClickNotif } from '@/hooks';
import { useNotificationStore } from '@/store';
import { ACTION, getStatusIcon } from '@/utils';
import { useOnClickOutside, useTimeAgo } from '@/hooks';
import theme, { hexPercentValues } from '@/styles/theme';
import { NoDataFound, Text } from '@/reuseable-components';
import type { Notification, NotificationType } from '@/types';

const Icon = styled.div`
  position: relative;
  width: 36px;
  height: 36px;
  border-radius: 100%;
  background-color: ${({ theme }) => theme.colors.white_opacity['008']};
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  &:hover {
    background-color: ${({ theme }) => theme.colors.white_opacity['20']};
  }
`;

const LiveBadge = styled.div`
  position: absolute;
  top: 8px;
  right: 8px;
  width: 6px;
  height: 6px;
  border-radius: 100%;
  background-color: ${({ theme }) => theme.colors.orange_og};
`;

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
  background: linear-gradient(0deg, #242424 0%, rgba(36, 36, 36, 0.64) 50%, rgba(36, 36, 36, 0) 100%);
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
  const { notifications, markAsSeen } = useNotificationStore();
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
      <Icon onClick={toggleOpen}>
        {!!unseenCount && <LiveBadge />}
        <Image src='/icons/common/notification.svg' alt='logo' width={16} height={16} />
      </Icon>

      {isOpen && (
        <AbsoluteContainer>
          <PopupHeader>
            <Text size={20}>Notifications</Text>{' '}
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
  background-color: ${({ theme }) => theme.colors.white_opacity['004']};
  cursor: not-allowed;
  &.click-enabled {
    cursor: pointer;
    &:hover {
      background-color: ${({ theme }) => theme.colors.white_opacity['008']};
    }
  }
`;

const StatusIcon = styled.div<{ $type: NotificationType }>`
  background-color: ${({ $type, theme }) => theme.text[$type] + hexPercentValues['012']};
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
  const { id, seen, type, title, message, time, crdType, target } = props;
  const canClick = !!crdType && !!target;

  const isDeleted = useMemo(() => {
    const deleteAction = ACTION.DELETE.toLowerCase(),
      titleIncludes = title?.toLowerCase().includes(deleteAction),
      messageIncludes = message?.toLowerCase().includes(deleteAction);

    return titleIncludes || messageIncludes || false;
  }, [title, message]);

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
      <StatusIcon $type={isDeleted ? 'error' : type}>
        <Image src={isDeleted ? '/icons/common/trash.svg' : getStatusIcon(type)} alt='status' width={16} height={16} />
      </StatusIcon>

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
