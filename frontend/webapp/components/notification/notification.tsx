import React, { useEffect, useState } from 'react';
import theme from '@/styles/palette';
import { useDispatch } from 'react-redux';
import styled, { keyframes } from 'styled-components';

import { BlueInfo, GreenCheck, RedError } from '@/assets/icons/app';
import { markAsOld, markAsSeen } from '@/store';
import { KeyvalLink, KeyvalText } from '@/design.system';
import { ROUTES } from '@/utils';
import { useRouter } from 'next/navigation';

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

const slideIn = keyframes`
  from {
    transform: translateX(100%);
  }
  to {
    transform: translateX(0);
  }
`;

const slideOut = keyframes`
  from {
    transform: translateX(0);
  }
  to {
    transform: translateX(100%);
  }
`;

const NotificationContainer = styled.div<{ type: string; isLeaving: boolean }>`
  border: 1px solid ${theme.colors.blue_grey};
  background-color: ${theme.colors.dark};
  border-radius: 8px;
  padding: 10px;
  gap: 12px;
  margin-top: 8px;
  max-width: 450px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  animation: ${(props) => (props.isLeaving ? slideOut : slideIn)} 0.5s forwards;

  &:hover {
    animation-play-state: paused;
  }
`;

const NotificationContent = styled.div`
  display: flex;
  gap: 8px;
`;

const NotificationDetails = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

const ButtonContainer = styled.div`
  display: flex;
  gap: 8px;
`;

const IconWrapper = styled.div<{ bgColor: string }>`
  width: 32px;
  height: 32px;
  border-radius: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: ${({ bgColor }) => bgColor};
`;

const InnerIconWrapper = styled.div<{ borderColor: string }>`
  width: 16px;
  height: 16px;
  border-radius: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid ${({ borderColor }) => borderColor};
`;

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

  const getSuccessIcon = () => (
    <IconWrapper bgColor="#3fb94f40">
      <InnerIconWrapper borderColor="#3fb950">
        <GreenCheck style={{ width: 10, height: 10 }} />
      </InnerIconWrapper>
    </IconWrapper>
  );

  const getErrorIcon = () => (
    <IconWrapper bgColor="#f8524952">
      <InnerIconWrapper borderColor="#f85249">
        <RedError
          style={{ width: 10, height: 10, marginLeft: 2, marginBottom: 2 }}
        />
      </InnerIconWrapper>
    </IconWrapper>
  );

  const getInfoIcon = () => (
    <IconWrapper bgColor="#2196F340">
      <InnerIconWrapper borderColor="#2196F3">
        <BlueInfo />
      </InnerIconWrapper>
    </IconWrapper>
  );

  const getIcon = () => {
    switch (type) {
      case 'success':
        return getSuccessIcon();
      case 'error':
        return getErrorIcon();
      case 'info':
        return getInfoIcon();
      default:
        return null;
    }
  };

  function onDetailsClick() {
    dispatch(markAsSeen(id));
    dispatch(markAsOld(id));

    if (target) {
      switch (crdType) {
        case 'Destination':
          router.push(`${ROUTES.UPDATE_DESTINATION}${target}`);
          break;
        case 'Source':
          router.push(`${ROUTES.MANAGE_SOURCE}?${target}`);
          break;
        default:
          break;
      }
    }
  }

  return (
    <NotificationContainer type={type} isLeaving={isLeaving}>
      <NotificationContent>
        <div>{getIcon()}</div>
        <NotificationDetails>
          <KeyvalText size={16} weight={600}>
            {title}
          </KeyvalText>
          <KeyvalText size={14}>{message}</KeyvalText>
        </NotificationDetails>
      </NotificationContent>
      <ButtonContainer>
        <KeyvalLink fontSize={12} value="Details" onClick={onDetailsClick} />
      </ButtonContainer>
    </NotificationContainer>
  );
};

export default Notification;
