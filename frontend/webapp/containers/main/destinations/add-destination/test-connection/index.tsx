import Image from 'next/image';
import styled from 'styled-components';
import React, { useState } from 'react';
import { DestinationInput } from '@/types';
import { useTestConnection } from '@/hooks';
import { Button, FadeLoader, Text } from '@/reuseable-components';

interface TestConnectionProps {
  destination: DestinationInput | undefined;
  onError?: () => void;
}

const ActionButton = styled(Button)<{ $isTestConnectionSuccess?: boolean }>`
  display: flex;
  align-items: center;
  gap: 8px;
  background-color: ${({ $isTestConnectionSuccess }) => ($isTestConnectionSuccess ? 'rgba(129, 175, 101, 0.16)' : 'transparent')};
`;

const ActionButtonText = styled(Text)<{ $isTestConnectionSuccess?: boolean }>`
  font-family: ${({ theme }) => theme.font_family.secondary};
  font-weight: 500;
  text-decoration: underline;
  text-transform: uppercase;
  font-size: 14px;
  line-height: 157.143%;
  color: ${({ theme, $isTestConnectionSuccess }) => ($isTestConnectionSuccess ? theme.text.success : theme.colors.white)};
`;

const TestConnection: React.FC<TestConnectionProps> = ({ destination, onError }) => {
  const [isTestConnectionSuccess, setIsTestConnectionSuccess] = useState<boolean>(false);
  const { testConnection, loading, error } = useTestConnection();

  const onButtonClick = async () => {
    if (!destination) {
      return;
    }

    const res = await testConnection(destination);
    if (res) {
      setIsTestConnectionSuccess(res.succeeded);
      !res.succeeded && onError && onError();
    }
  };
  return (
    <ActionButton variant={'secondary'} onClick={onButtonClick} $isTestConnectionSuccess={isTestConnectionSuccess}>
      {isTestConnectionSuccess && <Image alt='checkmark' src='/icons/common/connection-succeeded.svg' width={16} height={16} />}
      {loading && <FadeLoader />}

      <ActionButtonText size={14} $isTestConnectionSuccess={isTestConnectionSuccess}>
        {loading ? 'Checking' : isTestConnectionSuccess ? 'Connection ok' : 'Test Connection'}
      </ActionButtonText>
    </ActionButton>
  );
};

export { TestConnection };
