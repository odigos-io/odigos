import React from 'react';
import styled from 'styled-components';
import { Notification } from '@/types';
import { useNotificationStore } from '@/store';
import { NotificationNote } from '@/reuseable-components';
import { useClickNotif } from '@/hooks/notification/useClickNotif';

const Container = styled.div`
  position: fixed;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 10000;
  display: flex;
  flex-direction: column-reverse;
  gap: 6px;
  min-width: 600px;
`;

export const ToastList: React.FC = () => {
  const { notifications } = useNotificationStore();

  return (
    <Container>
      {notifications
        .filter(({ dismissed }) => !dismissed)
        .map((notif) => (
          <Toast key={`toast-${notif.id}`} {...notif} />
        ))}
    </Container>
  );
};

const Toast: React.FC<Notification> = (props) => {
  const { id, type, title, message, crdType, target } = props;
  const { markAsDismissed, markAsSeen } = useNotificationStore();
  const clickNotif = useClickNotif();

  const onClose = ({ asSeen }) => {
    markAsDismissed(id);
    if (asSeen) markAsSeen(id);
  };

  return (
    <NotificationNote
      id={id}
      type={type}
      title={title}
      message={message}
      action={
        crdType && target
          ? {
              label: 'go to details',
              onClick: () => clickNotif(props, { dismissToast: true }),
            }
          : undefined
      }
      onClose={onClose}
    />
  );
};
