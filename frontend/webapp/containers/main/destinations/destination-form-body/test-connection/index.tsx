import React, { useEffect } from 'react';
import Image from 'next/image';
import theme from '@/styles/theme';
import { getStatusIcon } from '@/utils';
import { useTestConnection } from '@/hooks';
import type { DestinationInput } from '@/types';
import styled, { css } from 'styled-components';
import { Button, FadeLoader, Text } from '@/reuseable-components';

type Status = 'success' | 'error';

interface Props {
  destination: DestinationInput;
  disabled: boolean;
  status?: Status;
  onError: () => void;
  onSuccess: () => void;
  validateForm: () => boolean;
}

const ActionButton = styled(Button)<{ $status?: Status }>`
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

  return (
    <ActionButton $status={status} variant='secondary' disabled={disabled} onClick={onClick}>
      {loading ? <FadeLoader /> : !!status ? <Image src={getStatusIcon(status)} alt='status' width={16} height={16} /> : null}

      <Text family='secondary' decoration='underline' size={14} color={!!status ? theme.text[status] : undefined}>
        {loading ? 'Checking' : status === 'success' ? 'Connection OK' : status === 'error' ? 'Connection Failed' : 'Test Connection'}
      </Text>
    </ActionButton>
  );
};
