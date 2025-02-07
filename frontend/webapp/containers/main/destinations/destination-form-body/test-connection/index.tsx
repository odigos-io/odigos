import React, { useEffect } from 'react';
import Theme from '@odigos/ui-theme';
import { useTestConnection } from '@/hooks';
import styled, { css } from 'styled-components';
import { type DestinationInput } from '@/types';
import { Button, FadeLoader, Text } from '@odigos/ui-components';
import { getStatusIcon, NOTIFICATION_TYPE } from '@odigos/ui-utils';

export type ConnectionStatus = NOTIFICATION_TYPE.SUCCESS | NOTIFICATION_TYPE.ERROR;

interface Props {
  destination: DestinationInput;
  disabled: boolean;
  status?: ConnectionStatus;
  onError: () => void;
  onSuccess: () => void;
  validateForm: () => boolean;
}

const ActionButton = styled(Button)<{ $status?: ConnectionStatus }>`
  display: flex;
  align-items: center;
  gap: 8px;

  ${({ $status, theme }) =>
    $status === 'success'
      ? css`
          border-color: transparent;
          background-color: ${theme.colors.success};
        `
      : $status === 'error'
      ? css`
          border-color: transparent;
          background-color: ${theme.colors.error};
        `
      : css`
          background-color: transparent;
        `}
`;

export const TestConnection: React.FC<Props> = ({ destination, disabled, status, onError, onSuccess, validateForm }) => {
  const theme = Theme.useTheme();
  const { testConnection, loading, data } = useTestConnection();

  useEffect(() => {
    if (data) {
      const { succeeded } = data.testConnectionForDestination;

      if (succeeded) onSuccess();
      else onError();
    }
  }, [data]);

  const onClick = async () => {
    if (validateForm()) await testConnection(destination);
  };

  const Icon = !!status ? getStatusIcon(status) : undefined;

  return (
    <ActionButton $status={status} variant='secondary' disabled={disabled} onClick={onClick}>
      {loading ? <FadeLoader /> : Icon ? <Icon /> : null}

      <Text family='secondary' decoration='underline' size={14} color={!!status ? theme.text[status] : undefined}>
        {loading ? 'Checking' : status === 'success' ? 'Connection OK' : status === 'error' ? 'Connection Failed' : 'Test Connection'}
      </Text>
    </ActionButton>
  );
};
