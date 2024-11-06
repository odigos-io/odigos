import React, { useCallback, useEffect, useState } from 'react';
import Image from 'next/image';
import { Text } from '../text';
import { progress, slide } from '@/styles';
import theme from '@/styles/theme';
import { Divider } from '../divider';
import styled from 'styled-components';
import { getStatusIcon } from '@/utils';
import { useNotificationStore } from '@/store';
import type { Notification, NotificationType } from '@/types';

interface NotificationProps {
  id?: string;
  type: NotificationType;
  title?: Notification['title'];
  message?: Notification['message'];
  action?: {
    label: string;
    onClick: () => void;
  };
  style?: React.CSSProperties;
}

const TOAST_DURATION = 5000;
const TRANSITION_DURATION = 500;

const Container = styled.div<{ isLeaving?: boolean }>`
  position: relative;
  &.animated {
    overflow: hidden;
    padding-bottom: 1px;
    border-radius: 32px;
    animation: ${({ isLeaving }) => (isLeaving ? slide.out['bottom'] : slide.in['bottom'])} ${TRANSITION_DURATION}ms forwards;
  }
`;

const DurationAnimation = styled.div<{ type: NotificationType }>`
  position: absolute;
  bottom: -1px;
  left: 0;
  z-index: -1;
  width: 100%;
  height: 100%;
  border-radius: 32px;
  background-color: ${({ type, theme }) => theme.text[type]};
  animation: ${progress.out} ${TOAST_DURATION - TRANSITION_DURATION}ms forwards;
`;

const Content = styled.div<{ type: NotificationType }>`
  display: flex;
  align-items: center;
  flex: 1;
  padding: 12px 16px;
  border-radius: 32px;
  background-color: ${({ type, theme }) => theme.colors[type]};
`;

const TextWrapper = styled.div`
  display: flex;
  align-items: center;
  margin: 0 auto 0 12px;
  height: 12px;
`;

const Title = styled(Text)<{ type: NotificationType }>`
  font-size: 14px;
  color: ${({ type, theme }) => theme.text[type]};
`;

const Message = styled(Text)<{ type: NotificationType }>`
  font-size: 12px;
  color: ${({ type, theme }) => theme.text[type]};
`;

const ButtonsWrapper = styled.div`
  margin-left: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
`;

const ActionButton = styled(Text)`
  text-transform: uppercase;
  text-decoration: underline;
  font-size: 14px;
  font-family: ${({ theme }) => theme.font_family.secondary};
  cursor: pointer;
`;

const CloseButton = styled(Image)`
  margin-left: 12px;
  width: 18px;
  height: 18px;
  padding: 4px;
  border-radius: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  &:hover {
    background-color: ${({ theme }) => theme.colors.white_opacity['10']};
  }
`;

const NotificationNote: React.FC<NotificationProps> = ({ id, type, title, message, action, style }) => {
  const { markAsDismissed, markAsSeen } = useNotificationStore();
  const [isEntering, setIsEntering] = useState(true);
  const [isLeaving, setIsLeaving] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => setIsEntering(false), TRANSITION_DURATION);
    return () => clearTimeout(timer);
  }, []);

  const closeToast = useCallback(
    (params?: { asSeen: boolean }) => {
      if (!!id) {
        setIsLeaving(true);

        const timer = setTimeout(() => {
          if (params?.asSeen) markAsSeen(id);
          markAsDismissed(id);
          clearTimeout(timer);
        }, TRANSITION_DURATION);
      }
    },
    [id]
  );

  useEffect(() => {
    const timer = setTimeout(closeToast, TOAST_DURATION);
    return () => clearTimeout(timer);
  }, [closeToast]);

  return (
    <Container className={id ? 'animated' : ''} isLeaving={isLeaving}>
      <Content type={type} style={style}>
        <Image src={getStatusIcon(type)} alt={type} width={16} height={16} />

        <TextWrapper>
          {title && <Title type={type}>{title}</Title>}
          {title && message && <Divider orientation='vertical' color={theme.text[type] + '4D'} thickness={1} />}
          {message && <Message type={type}>{message}</Message>}
        </TextWrapper>

        {(action || id) && (
          <ButtonsWrapper>
            {action && <ActionButton onClick={action.onClick}>{action.label}</ActionButton>}
            {id && <CloseButton src='/icons/common/x.svg' alt='x' width={12} height={12} onClick={() => closeToast({ asSeen: true })} />}
          </ButtonsWrapper>
        )}
      </Content>

      {id && !isEntering && <DurationAnimation type={type} />}
    </Container>
  );
};

export { NotificationNote };
