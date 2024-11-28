import React, { useEffect, useMemo } from 'react';
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
  validateForm: () => boolean;
  onError: () => void;
  onSuccess: () => void;
  clearStatus: () => void;
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
          // border-color: transparent;
          background-color: transparent;
        `}
`;

export const TestConnection: React.FC<Props> = ({ destination, disabled, validateForm, onError, onSuccess, clearStatus }) => {
  const { testConnection, loading, data } = useTestConnection();

  useEffect(() => {
    clearStatus();

    if (data) {
      const { succeeded } = data.testConnectionForDestination;

      if (succeeded) onSuccess();
      else onError();
    }
  }, [data]);

  const status: Status | undefined = useMemo(() => {
    if (!data) return undefined;

    const { succeeded } = data.testConnectionForDestination;

    if (succeeded) return 'success';
    else return 'error';
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
