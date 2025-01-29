import React from 'react';
import { getStatusIcon } from '@/utils';
import { NOTIFICATION_TYPE } from '@/types';
import styled, { useTheme } from 'styled-components';
import { Divider, Text, Theme } from '@odigos/ui-components';

export * from './active-status';
export * from './connection-status';
export * from './instrument-status';

export interface StatusProps {
  title?: string;
  subtitle?: string;
  size?: number;
  family?: 'primary' | 'secondary';
  status?: NOTIFICATION_TYPE;
  isActive?: boolean;
  withIcon?: boolean;
  withBorder?: boolean;
  withBackground?: boolean;
}

const StatusWrapper = styled.div<{
  $size: number;
  $status: NOTIFICATION_TYPE;
  $withIcon?: StatusProps['withIcon'];
  $withBorder?: StatusProps['withBorder'];
  $withBackground?: StatusProps['withBackground'];
}>`
  display: flex;
  align-items: center;
  gap: ${({ $size }) => $size / 3}px;
  padding: ${({ $size, $withBorder, $withBackground }) => ($withBorder || $withBackground ? `${$size / ($withBorder ? 3 : 2)}px ${$size / ($withBorder ? 1.5 : 1)}px` : '0')};
  width: fit-content;
  border-radius: 360px;
  border: ${({ $withBorder, $status, theme }) => ($withBorder ? `1px solid ${theme.text[$status] + Theme.hexPercent['050']}` : 'none')};
  background: ${({ $withBackground, $status, theme }) => ($withBackground ? `linear-gradient(90deg, transparent 0%, ${theme.text[$status] + Theme.hexPercent['030']} 100%)` : 'transparent')};
`;

const IconWrapper = styled.div`
  display: flex;
  align-items: center;
`;

const TextWrapper = styled.div`
  display: flex;
  align-items: center;
`;

export const Status: React.FC<StatusProps> = ({ title, subtitle, size = 12, family = 'secondary', status, isActive: oldStatus, withIcon, withBorder, withBackground }) => {
  const statusType = typeof oldStatus === 'boolean' ? (oldStatus ? NOTIFICATION_TYPE.SUCCESS : NOTIFICATION_TYPE.ERROR) : status || NOTIFICATION_TYPE.DEFAULT;
  const StatusIcon = getStatusIcon(statusType);
  const theme = useTheme();

  return (
    <StatusWrapper $size={size} $status={statusType} $withIcon={withIcon} $withBorder={withBorder} $withBackground={withBackground}>
      {withIcon && (
        <IconWrapper>
          <StatusIcon size={size + 2} />
        </IconWrapper>
      )}

      {(!!title || !!subtitle) && (
        <TextWrapper>
          {!!title && (
            <Text size={size} family={family} color={theme.text[statusType]}>
              {title}
            </Text>
          )}

          {!!subtitle && (
            <TextWrapper>
              <Divider orientation='vertical' length={`${size - 2}px`} type={statusType} />
              <Text size={size - 2} family={family} color={theme.text[`${statusType}_secondary`]}>
                {subtitle}
              </Text>
            </TextWrapper>
          )}
        </TextWrapper>
      )}
    </StatusWrapper>
  );
};
