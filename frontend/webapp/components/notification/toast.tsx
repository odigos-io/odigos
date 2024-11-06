import React, { useEffect, useState } from 'react';
import type { Notification } from '@/types';
import { useNotificationStore } from '@/store';
import { NotificationNote } from '@/reuseable-components';

interface Props extends Notification {}

const Toast: React.FC<Props> = ({ id, message, type, title, crdType, target }) => {
  // const router = useRouter();
  const { markAsDismissed, markAsSeen } = useNotificationStore();
  const [isLeaving, setIsLeaving] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => {
      setIsLeaving(true);
      setTimeout(() => markAsDismissed(id), 500);
    }, 5000);

    return () => clearTimeout(timer);
  }, [id, markAsDismissed]);

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

export default Toast;
