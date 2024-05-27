import React from 'react';
import styled from 'styled-components';
import { useDispatch } from 'react-redux';
import { markAsSeen, removeNotification } from '@/store';
import { GreenCheck, RedError } from '@/assets/icons/app';
import theme from '@/styles/palette';

interface NotificationListItemProps {
  id: string;
  message: string;
  type: 'success' | 'error' | 'info';
  seen: boolean;
}

const NotificationItemContainer = styled.div<{ seen: boolean }>`
  border-bottom: 1px solid ${theme.colors.blue_grey};
  padding: 10px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: ${({ seen }) => (seen ? 'white' : theme.colors.light_dark)};
`;

const NotificationContent = styled.div`
  display: flex;
  gap: 8px;
`;

const ConditionIconContainer = styled.div`
  display: flex;
  align-items: center;
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

const ButtonsContainer = styled.div`
  display: flex;
  gap: 8px;
`;

const Button = styled.button`
  background: transparent;
  border: none;
  color: ${theme.text.primary};
  cursor: pointer;

  &:hover {
    color: ${theme.text.secondary};
  }
`;

const NotificationListItem: React.FC<NotificationListItemProps> = ({
  id,
  message,
  type,
  seen,
}) => {
  const dispatch = useDispatch();

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
        <i
          className="fa fa-info"
          style={{ color: '#2196F3', fontSize: '10px' }}
        ></i>
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

  return (
    <NotificationItemContainer seen={seen}>
      <NotificationContent>
        <ConditionIconContainer>{getIcon()}</ConditionIconContainer>
        <div>
          <div>{message}</div>
        </div>
      </NotificationContent>
      <ButtonsContainer>
        <Button onClick={() => dispatch(markAsSeen(id))}>Seen</Button>
        <Button onClick={() => dispatch(removeNotification(id))}>Close</Button>
      </ButtonsContainer>
    </NotificationItemContainer>
  );
};

export default NotificationListItem;
