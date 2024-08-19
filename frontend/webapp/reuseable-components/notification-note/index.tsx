import React from 'react';
import styled, { css } from 'styled-components';
import Image from 'next/image';
import { Text } from '../text';

// Define the notification types
type NotificationType = 'warning' | 'error' | 'success' | 'info';

interface NotificationProps {
  type: NotificationType;
  text: string;
}

const NotificationContainer = styled.div<{ type: NotificationType }>`
  display: flex;
  align-items: center;
  padding: 12px 16px;
  border-radius: 32px;

  background-color: ${({ type }) => {
    switch (type) {
      case 'warning':
        return '#472300'; // Orange
      case 'error':
        return '#FF4C4C'; // Red
      case 'success':
        return '#28A745'; // Green
      case 'info':
        return '#2B2D66'; // Blue
      default:
        return '#2B2D66'; // Default to info color
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
        return '#E9CF35'; // Orange
      case 'error':
        return '#FF4C4C'; // Red
      case 'success':
        return '#28A745'; // Green
      case 'info':
        return '#2B2D66'; // Blue
      default:
        return '#2B2D66'; // Default to info color
    }
  }};
`;

// Icons can be dynamically rendered based on the type
const NotificationIcon = ({ type }: { type: NotificationType }) => {
  switch (type) {
    case 'warning':
      return (
        <Image
          src="/icons/notification/warning-icon.svg"
          alt="warning"
          width={16}
          height={16}
        />
      );
    case 'error':
      return (
        <Image
          src="/icons/notification/error-icon.svg"
          alt="error"
          width={16}
          height={16}
        />
      );
    case 'success':
      return (
        <Image
          src="/icons/notification/success-icon.svg"
          alt="success"
          width={16}
          height={16}
        />
      );
    case 'info':
    default:
      return (
        <Image src="/icons/info-icon.svg" alt="info" width={16} height={16} />
      );
  }
};

const NotificationNote: React.FC<NotificationProps> = ({ type, text }) => {
  return (
    <NotificationContainer type={type}>
      <IconWrapper>
        <NotificationIcon type={type} />
      </IconWrapper>
      <Title type={type}>{text}</Title>
    </NotificationContainer>
  );
};

export { NotificationNote };
