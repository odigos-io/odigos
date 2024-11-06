import React from 'react';
import Image from 'next/image';
import { Text } from '../text';
import { Divider } from '../divider';
import styled from 'styled-components';
import type { Notification, NotificationType } from '@/types';

interface NotificationProps {
  type: NotificationType;
  title: Notification['title'];
  message: Notification['message'];
  action?: {
    label: string;
    onClick: () => void;
  };
  style?: React.CSSProperties;
}

const getTextColor = ({ type }: { type: NotificationType }) => {
  switch (type) {
    case 'warning':
      return '#E9CF35';
    case 'error':
      return '#E25A5A';
    case 'success':
      return '#81AF65';
    case 'info':
      return '#B8B8B8';
    case 'default':
    default:
      return '#AABEF7';
  }
};

const getBackgroundColor = ({ type }: { type: NotificationType }) => {
  switch (type) {
    case 'warning':
      return '#472300';
    case 'error':
      return '#431919';
    case 'success':
      return '#172013';
    case 'info':
      return '#242424';
    case 'default':
    default:
      return '#181944';
  }
};

const getIconSource = ({ type }: { type: NotificationType }) => {
  switch (type) {
    case 'warning':
      return '/icons/notification/warning-icon.svg';
    case 'error':
      return '/icons/notification/error-icon.svg';
    case 'success':
      return '/icons/notification/success-icon.svg';
    case 'info':
      return '/icons/common/info.svg';
    default:
      return '/brand/odigos-icon.svg';
  }
};

const NotificationContainer = styled.div<{ type: NotificationType }>`
  display: flex;
  align-items: center;
  padding: 12px 16px;
  border-radius: 32px;
  background-color: ${getBackgroundColor};
`;

const TextWrapper = styled.div`
  display: flex;
  align-items: center;
  margin: 0 12px;
  height: 12px;
`;

const Title = styled(Text)<{ type: NotificationType }>`
  font-size: 14px;
  color: ${getTextColor};
`;

const Message = styled(Text)<{ type: NotificationType }>`
  font-size: 12px;
  color: ${getTextColor};
`;

const ActionButtonWrapper = styled.div`
  margin-left: 40px;
`;

const ActionButton = styled(Text)`
  text-transform: uppercase;
  text-decoration: underline;
  font-size: 14px;
  font-family: ${({ theme }) => theme.font_family.secondary};
  cursor: pointer;
`;

const NotificationNote: React.FC<NotificationProps> = ({ type, title, message, action, style }) => {
  return (
    <NotificationContainer type={type} style={style}>
      <Image src={getIconSource({ type })} alt={type} width={16} height={16} />

      <TextWrapper>
        {title && <Title type={type}>{title}</Title>}
        {title && message && <Divider orientation='vertical' color={getTextColor({ type }) + '4D'} thickness={1} />}
        {message && <Message type={type}>{message}</Message>}
      </TextWrapper>

      {action && (
        <ActionButtonWrapper onClick={action.onClick}>
          <ActionButton>{action.label}</ActionButton>
        </ActionButtonWrapper>
      )}
    </NotificationContainer>
  );
};

export { NotificationNote };
