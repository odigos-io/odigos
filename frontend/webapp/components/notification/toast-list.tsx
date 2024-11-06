import React from 'react';
import Toast from './toast';
import styled from 'styled-components';
import { useNotificationStore } from '@/store';

const Container = styled.div`
  position: fixed;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 10000;
  display: flex;
  flex-direction: column;
  gap: 6px;
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
