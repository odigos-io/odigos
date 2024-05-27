import React from 'react';
import styled from 'styled-components';
import { useDispatch } from 'react-redux';
import { markAsOld, markAsSeen } from '@/store';
import { GreenCheck, RedError } from '@/assets/icons/app';
import theme from '@/styles/palette';
import { KeyvalLink, KeyvalText } from '@/design.system';
import { timeAgo } from '@/utils';

interface NotificationListItemProps {
  id: string;
  message: string;
  type: 'success' | 'error' | 'info';
  seen: boolean;
  title?: string;
  onClick?: () => void;
  time?: string;
}

const NotificationItemContainer = styled.div<{ seen: boolean }>`
  border-bottom: 1px solid ${theme.colors.blue_grey};
  padding: 10px;
  gap: 12px;

  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: ${({ seen }) =>
    seen ? theme.colors.light_dark : theme.colors.dark};
`;

const NotificationContent = styled.div`
  display: flex;
  width: 300px;
  gap: 8px;
`;

const NotificationDetails = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
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

const ButtonContainer = styled.div`
  display: flex;
  gap: 8px;
`;

const NotificationListItem: React.FC<NotificationListItemProps> = ({
  id,
  message,
  type,
  seen,
  title,
  onClick = () => {},
  time,
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
  function onDetailsClick() {
    dispatch(markAsSeen(id));
    dispatch(markAsOld(id));
    onClick();
  }
  return (
    <NotificationItemContainer seen={seen}>
      <NotificationContent>
        <div>{getIcon()}</div>
        <NotificationDetails>
          <KeyvalText size={18} weight={600}>
            {title}
          </KeyvalText>
          <KeyvalText size={14}>{message}</KeyvalText>
          {time && (
            <KeyvalText color={theme.text.light_grey} size={12}>
              {timeAgo(time)}
            </KeyvalText>
          )}
        </NotificationDetails>
      </NotificationContent>
      <ButtonContainer>
        <KeyvalLink fontSize={12} value="Details" onClick={onDetailsClick} />
      </ButtonContainer>
    </NotificationItemContainer>
  );
};

export default NotificationListItem;
