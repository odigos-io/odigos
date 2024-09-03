import Image from 'next/image';
import React from 'react';
import { Text } from '../text';
import styled from 'styled-components';

interface ConnectionStatusProps {
  title: string;
  subtitle?: string;
  status: 'alive' | 'lost';
}

const StatusWrapper = styled.div<{ status: 'alive' | 'lost' }>`
  display: flex;
  align-items: center;
  padding: 8px 24px;
  border-radius: 32px;
  background: ${({ status }) =>
    status === 'alive'
      ? `linear-gradient(
    90deg,
    rgba(23, 32, 19, 0) 0%,
    rgba(23, 32, 19, 0.8) 50%,
    #172013 100%
  )`
      : `linear-gradient(90deg, rgba(51, 21, 21, 0.00) 0%, rgba(51, 21, 21, 0.80) 50%, #331515 100%)`};
`;

const IconWrapper = styled.div`
  margin-right: 8px;
  display: flex;
  align-items: center;
`;

const TextWrapper = styled.div`
  display: flex;
  align-items: center;
`;

const Title = styled(Text)<{ status: 'alive' | 'lost' }>`
  font-weight: 400;
  color: ${({ status, theme }) =>
    status === 'alive' ? theme.text.success : theme.text.error};
`;

const Subtitle = styled(Text)`
  font-size: 12px;
  font-weight: 400;
  color: #db5151;
`;

const TextDivider = styled.div`
  width: 1px;
  height: 12px;
  background: rgba(237, 124, 124, 0.16);
  margin: 0 8px;
`;

const ConnectionStatus: React.FC<ConnectionStatusProps> = ({
  title,
  subtitle,
  status,
}) => {
  return (
    <StatusWrapper status={status}>
      <IconWrapper>
        <Image
          src={`/icons/notification/${
            status === 'alive' ? 'success-icon' : 'error-icon2'
          }.svg`}
          alt="status"
          width={16}
          height={16}
        />
      </IconWrapper>
      <TextWrapper>
        <Title status={status}>{title}</Title>
        {subtitle && (
          <TextWrapper>
            <TextDivider />
            <Subtitle>{subtitle}</Subtitle>
          </TextWrapper>
        )}
      </TextWrapper>
    </StatusWrapper>
  );
};

export { ConnectionStatus };
