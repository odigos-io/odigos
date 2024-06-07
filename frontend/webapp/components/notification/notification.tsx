import React, { useEffect, useState } from 'react';
import { ROUTES } from '@/utils';
import { useDispatch } from 'react-redux';
import { useRouter } from 'next/navigation';
import { getIcon } from './notification-icon';
import { markAsOld, markAsSeen } from '@/store';
import { KeyvalLink, KeyvalText } from '@/design.system';
import {
  NotificationContainer,
  NotificationContentWrapper,
  NotificationDetailsWrapper,
  NotificationButtonContainer,
} from './notification-container';

interface NotificationProps {
  id: string;
  message: string;
  title?: string;
  type: 'success' | 'error' | 'info';
  time?: string;
  onClick?: () => void;
  crdType?: string;
  target?: string;
}

const Notification: React.FC<NotificationProps> = ({
  id,
  message,
  type,
  title,
  crdType,
  target,
}) => {
  const dispatch = useDispatch();
  const router = useRouter();
  const [isLeaving, setIsLeaving] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => {
      setIsLeaving(true);
      setTimeout(() => dispatch(markAsOld(id)), 500);
    }, 5000);

    return () => clearTimeout(timer);
  }, [id, dispatch]);

  function onDetailsClick() {
    dispatch(markAsSeen(id));
    dispatch(markAsOld(id));

    if (target) {
      switch (crdType) {
        case 'Destination':
          router.push(`${ROUTES.UPDATE_DESTINATION}${target}`);
          break;
        case 'InstrumentedApplication':
        case 'InstrumentationInstance':
          router.push(`${ROUTES.MANAGE_SOURCE}?${target}`);
          break;
        default:
          break;
      }
    }
  }

  return (
    <NotificationContainer type={type} isLeaving={isLeaving}>
      <NotificationContentWrapper>
        <div>{getIcon(type)}</div>
        <NotificationDetailsWrapper>
          <KeyvalText size={16} weight={600}>
            {title}
          </KeyvalText>
          <KeyvalText size={14}>{message}</KeyvalText>
        </NotificationDetailsWrapper>
      </NotificationContentWrapper>
      <NotificationButtonContainer>
        {!!target && (
          <KeyvalLink fontSize={12} value="Details" onClick={onDetailsClick} />
        )}
      </NotificationButtonContainer>
    </NotificationContainer>
  );
};

export default Notification;
