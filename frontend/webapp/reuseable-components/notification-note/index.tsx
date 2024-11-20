import React, { useCallback, useEffect, useRef, useState } from 'react';
import Image from 'next/image';
import { Text } from '../text';
import theme from '@/styles/theme';
import { Divider } from '../divider';
import styled from 'styled-components';
import { getStatusIcon } from '@/utils';
import { progress, slide } from '@/styles';
import type { Notification, NotificationType } from '@/types';

interface OnCloseParams {
  asSeen: boolean;
}

interface NotificationProps {
  id?: string;
  type: NotificationType;
  title?: Notification['title'];
  message?: Notification['message'];
  action?: {
    label: string;
    onClick: () => void;
  };
  onClose?: (params: OnCloseParams) => void;
  style?: React.CSSProperties;
}

const TOAST_DURATION = 5000;
const TRANSITION_DURATION = 500;

const Container = styled.div<{ $isLeaving?: boolean }>`
  position: relative;
  &.animated {
    overflow: hidden;
    padding-bottom: 1px;
    border-radius: 32px;
    animation: ${({ $isLeaving }) => ($isLeaving ? slide.out['bottom'] : slide.in['bottom'])} ${TRANSITION_DURATION}ms forwards;
  }
`;

const DurationAnimation = styled.div<{ $type: NotificationType }>`
  position: absolute;
  bottom: -1px;
  left: 0;
  z-index: -1;
  width: 100%;
  height: 100%;
  border-radius: 32px;
  background-color: ${({ $type, theme }) => theme.text[$type]};
  animation: ${progress.out} ${TOAST_DURATION - TRANSITION_DURATION}ms forwards;
`;

const Content = styled.div<{ $type: NotificationType }>`
  display: flex;
  align-items: center;
  flex: 1;
  padding: 12px 16px;
  border-radius: 32px;
  background-color: ${({ $type, theme }) => theme.colors[$type]};
`;

const TextWrapper = styled.div`
  display: flex;
  align-items: center;
  margin: 0 auto 0 12px;
  height: 12px;
`;

const Title = styled(Text)<{ $type: NotificationType }>`
  font-size: 14px;
  color: ${({ $type, theme }) => theme.text[$type]};
`;

const Message = styled(Text)<{ $type: NotificationType }>`
  font-size: 12px;
  color: ${({ $type, theme }) => theme.text[$type]};
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

const NotificationNote: React.FC<NotificationProps> = ({ type, title, message, action, onClose, style }) => {
  // These are for handling transitions:
  // isEntering - to stop the progress bar from rendering before the toast is fully slide-in
  // isLeaving - to trigger the slide-out animation
  const [isEntering, setIsEntering] = useState(true);
  const [isLeaving, setIsLeaving] = useState(false);

  // These are for handling on-hover events (pause/resume the progress bar animation & timeout for auto-close/dismiss)
  const timerForClosure = useRef<NodeJS.Timeout | null>(null);
  const progress = useRef<HTMLDivElement | null>(null);

  const closeToast = useCallback(
    (params: OnCloseParams) => {
      if (onClose) {
        setIsLeaving(true);
        setTimeout(() => {
          onClose({ asSeen: params?.asSeen });
        }, TRANSITION_DURATION);
      }
    },
    [onClose],
  );

  useEffect(() => {
    const t = setTimeout(() => setIsEntering(false), TRANSITION_DURATION);

    return () => {
      clearTimeout(t);
    };
  }, []);

  useEffect(() => {
    timerForClosure.current = setTimeout(() => closeToast({ asSeen: false }), TOAST_DURATION);

    return () => {
      if (timerForClosure.current) clearTimeout(timerForClosure.current);
    };
  }, []);

  const handleMouseEnter = () => {
    if (timerForClosure.current) clearTimeout(timerForClosure.current);
    if (progress.current) progress.current.style.animationPlayState = 'paused';
  };

  const handleMouseLeave = () => {
    if (progress.current) {
      const remainingTime = (progress.current.offsetWidth / (progress.current.parentElement as HTMLDivElement).offsetWidth) * 4000;

      timerForClosure.current = setTimeout(() => closeToast({ asSeen: false }), remainingTime);
      progress.current.style.animationPlayState = 'running';
    }
  };

  return (
    <Container className={onClose ? 'animated' : ''} $isLeaving={isLeaving} onMouseEnter={handleMouseEnter} onMouseLeave={handleMouseLeave}>
      <Content $type={type} style={style}>
        <Image src={getStatusIcon(type)} alt={type} width={16} height={16} />

        <TextWrapper>
          {title && <Title $type={type}>{title}</Title>}
          {title && message && <Divider orientation='vertical' color={theme.text[type] + '4D'} thickness={1} />}
          {message && <Message $type={type}>{message}</Message>}
        </TextWrapper>

        {(action || onClose) && (
          <ButtonsWrapper>
            {action && <ActionButton onClick={action.onClick}>{action.label}</ActionButton>}
            {onClose && <CloseButton src='/icons/common/x.svg' alt='x' width={12} height={12} onClick={() => closeToast({ asSeen: true })} />}
          </ButtonsWrapper>
        )}
      </Content>

      {onClose && !isEntering && !isLeaving && <DurationAnimation ref={progress} $type={type} />}
    </Container>
  );
};

export { NotificationNote };
