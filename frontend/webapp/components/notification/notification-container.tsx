import React from 'react';
import styled, { keyframes } from 'styled-components';
import theme from '@/styles/palette';

interface NotificationContainerProps {
  children: React.ReactNode;
  type?: string;
  isLeaving?: boolean;
  seen?: boolean;
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

const StyledNotificationContainer = styled.div<NotificationContainerProps>`
  border: 1px solid ${theme.colors.blue_grey};
  background-color: ${theme.colors.dark};
  border-radius: 8px;
  padding: 10px;
  gap: 12px;
  margin-top: 8px;
  width: 400px;
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

export const NotificationContainer: React.FC<NotificationContainerProps> = ({
  children,
  type,
  isLeaving,
  seen,
}) => (
  <StyledNotificationContainer type={type} isLeaving={isLeaving} seen={seen}>
    {children}
  </StyledNotificationContainer>
);

export const NotificationContentWrapper = NotificationContent;
export const NotificationDetailsWrapper = NotificationDetails;
export const NotificationButtonContainer = ButtonContainer;
