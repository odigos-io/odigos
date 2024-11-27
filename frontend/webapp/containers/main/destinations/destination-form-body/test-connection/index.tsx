import React, { useEffect, useMemo } from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { getStatusIcon } from '@/utils';
import { useTestConnection } from '@/hooks';
import type { DestinationInput } from '@/types';
import { Button, FadeLoader, Text } from '@/reuseable-components';

interface TestConnectionProps {
  destination: DestinationInput;
  disabled: boolean;
  clearStatus: () => void;
  onError: () => void;
}

const ActionButton = styled(Button)<{ $success?: boolean }>`
  display: flex;
  align-items: center;
  gap: 8px;
  background-color: ${({ $success }) => ($success ? 'rgba(129, 175, 101, 0.16)' : 'transparent')};
`;

const ActionButtonText = styled(Text)<{ $success?: boolean }>`
  font-family: ${({ theme }) => theme.font_family.secondary};
  font-weight: 500;
  text-decoration: underline;
  text-transform: uppercase;
  font-size: 14px;
  line-height: 157.143%;
  color: ${({ theme, $success }) => ($success ? theme.text.success : theme.colors.white)};
`;

export const TestConnection: React.FC<TestConnectionProps> = ({ destination, disabled, clearStatus, onError }) => {
  const { testConnection, loading, data } = useTestConnection();
  const success = useMemo(() => data?.testConnectionForDestination.succeeded || false, [data]);

  useEffect(() => {
    if (data) {
      clearStatus();
      if (!success) onError && onError();
    }
  }, [data, success]);

  return (
    <ActionButton variant='secondary' disabled={disabled} onClick={() => testConnection(destination)} $success={success}>
      {loading ? <FadeLoader /> : success ? <Image alt='checkmark' src={getStatusIcon('success')} width={16} height={16} /> : null}

      <ActionButtonText size={14} $success={success}>
        {loading ? 'Checking' : success ? 'Connection OK' : 'Test Connection'}
      </ActionButtonText>
    </ActionButton>
  );
};
