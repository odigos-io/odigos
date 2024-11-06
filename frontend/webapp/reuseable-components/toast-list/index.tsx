import React from 'react';
import styled from 'styled-components';
import { Notification } from '@/types';
import { useNotificationStore } from '@/store';
import { NotificationNote } from '@/reuseable-components';

const Container = styled.div`
  position: fixed;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 10000;
  display: flex;
  flex-direction: column;
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

const Toast: React.FC<Notification> = ({ id, type, title, message, crdType, target }) => {
  const { markAsDismissed, markAsSeen } = useNotificationStore();
  // const router = useRouter();

  const onClick = () => {
    markAsDismissed(id);
    markAsSeen(id);

    alert('TODO');

    // switch (crdType) {
    //   case 'Destination':
    //     // TODO: open drawer
    //     // router.push(`${ROUTES.UPDATE_DESTINATION}${target}`);
    //     break;
    //   case 'InstrumentedApplication':
    //   case 'InstrumentationInstance':
    //     // TODO: open drawer
    //     // router.push(`${ROUTES.MANAGE_SOURCE}?${target}`);
    //     break;
    //   default:
    //     break;
    // }
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
              onClick,
            }
          : undefined
      }
    />
  );
};
