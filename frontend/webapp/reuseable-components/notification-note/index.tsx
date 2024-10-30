import React from 'react';
import styled, { css } from 'styled-components';
import Image from 'next/image';
import { Text } from '../text';

// Define the notification types
type NotificationType = 'warning' | 'error' | 'success' | 'info' | 'default';

interface NotificationProps {
  type: NotificationType;
  text: string;
  action?: {
    label: string;
    onClick: () => void;
  };
  style?: React.CSSProperties;
}

const NotificationContainer = styled.div<{ type: NotificationType }>`
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-radius: 32px;

  background-color: ${({ type }) => {
    switch (type) {
      case 'warning':
        return '#472300'; // Orange
      case 'error':
        return 'rgba(226, 90, 90, 0.12);';
      case 'success':
        return '#28A745'; // Green
      case 'info':
        return '#F9F9F90A'; // Default to info color
      case 'default':
      default:
        return '#181944'; // Blue
    }
  }};
`;

const IconWrapper = styled.div`
  margin-right: 12px;
  display: flex;
  justify-content: center;
  align-items: center;
`;

const Title = styled(Text)<{ type: NotificationType }>`
  font-size: 14px;
  color: ${({ type }) => {
    switch (type) {
      case 'warning':
        return '#E9CF35';
      case 'error':
        return '#E25A5A';
      case 'success':
        return '#28A745';
      case 'info':
        return '#B8B8B8';
      case 'default':
      default:
        return '#AABEF7';
    }
  }};
`;

const TitleWrapper = styled.div`
  display: flex;
  align-items: center;
`;

const ActionButtonWrapper = styled.div`
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
`;

const ActionButton = styled(Text)`
  text-decoration: underline;
  text-transform: uppercase;
  font-size: 14px;
  font-weight: 400;
  font-family: ${({ theme }) => theme.font_family.secondary};
`;

const NotificationIcon = ({ type }: { type: NotificationType }) => {
  switch (type) {
    case 'warning':
      return <Image src='/icons/notification/warning-icon.svg' alt='warning' width={16} height={16} />;
    case 'error':
      return <Image src='/icons/notification/error-icon.svg' alt='error' width={16} height={16} />;
    case 'success':
      return <Image src='/icons/notification/success-icon.svg' alt='success' width={16} height={16} />;
    case 'info':
      return <Image src='/icons/common/info.svg' alt='info' width={16} height={16} />;
    default:
      return <Image src='/brand/odigos-icon.svg' alt='info' width={16} height={16} />;
  }
};

const NotificationNote: React.FC<NotificationProps> = ({ type, text, action, style }) => {
  return (
    <NotificationContainer type={type} style={style}>
      <TitleWrapper>
        <IconWrapper>
          <NotificationIcon type={type} />
        </IconWrapper>
        <Title type={type}>{text}</Title>
      </TitleWrapper>
      {action && (
        <ActionButtonWrapper onClick={action.onClick}>
          <ActionButton decoration='under'>{action.label}</ActionButton>
        </ActionButtonWrapper>
      )}
    </NotificationContainer>
  );
};

export { NotificationNote };
