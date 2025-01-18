import React from 'react';
import styled from 'styled-components';
import { getStatusIcon } from '@/utils';
import { NOTIFICATION_TYPE } from '@/types';
import { Divider, Text } from '@/reuseable-components';
import theme, { hexPercentValues } from '@/styles/theme';
import { CheckCircledIcon, CrossCircledIcon } from '@/assets';

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
  isPale?: boolean;
  withIcon?: boolean;
  withBorder?: boolean;
  withBackground?: boolean;
}

const StatusWrapper = styled.div<{
  $size: number;
  $isPale: StatusProps['isPale'];
  $status: StatusProps['status'];
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
  border: ${({ $withBorder, $isPale, $status, theme }) =>
    $withBorder
      ? `1px solid ${
          $isPale ? theme.colors.border : $status === NOTIFICATION_TYPE.SUCCESS ? theme.colors.dark_green : $status === NOTIFICATION_TYPE.ERROR ? theme.colors.dark_red : theme.colors.border
        }`
      : 'none'};
  background: ${({ $withBackground, $isPale, $status = NOTIFICATION_TYPE.DEFAULT, theme }) =>
    $withBackground
      ? $isPale
        ? `linear-gradient(90deg, transparent 0%, ${theme.colors.info + hexPercentValues['080']} 50%, ${theme.colors.info} 100%)`
        : `linear-gradient(90deg, transparent 0%, ${theme.colors[$status] + hexPercentValues['080']} 50%, ${theme.colors[$status]} 100%)`
      : 'transparent'};
`;

const IconWrapper = styled.div`
  display: flex;
  align-items: center;
`;

const TextWrapper = styled.div`
  display: flex;
  align-items: center;
`;

const Title = styled(Text)<{
  $isPale: StatusProps['isPale'];
  $status: StatusProps['status'];
}>`
  color: ${({ $isPale, $status = NOTIFICATION_TYPE.DEFAULT, theme }) => ($isPale ? theme.text.secondary : theme.text[$status])};
`;

const SubTitle = styled(Text)<{
  $isPale: StatusProps['isPale'];
  $status: StatusProps['status'];
}>`
  color: ${({ $isPale, $status = NOTIFICATION_TYPE.DEFAULT }) => ($isPale ? theme.text.grey : theme.text[`${$status}_secondary`])};
`;

export const Status: React.FC<StatusProps> = ({ title, subtitle, size = 12, family = 'secondary', status, isActive: oldStatus, isPale, withIcon, withBorder, withBackground }) => {
  const statusType = status || (oldStatus ? NOTIFICATION_TYPE.SUCCESS : NOTIFICATION_TYPE.ERROR);
  const StatusIcon = getStatusIcon(statusType);

  return (
    <StatusWrapper $size={size} $isPale={isPale} $status={statusType} $withIcon={withIcon} $withBorder={withBorder} $withBackground={withBackground}>
      {withIcon && (
        <IconWrapper>
          {isPale && statusType === NOTIFICATION_TYPE.SUCCESS ? (
            <CheckCircledIcon size={size + 2} />
          ) : isPale && statusType === NOTIFICATION_TYPE.ERROR ? (
            <CrossCircledIcon size={size + 2} />
          ) : (
            <StatusIcon size={size + 2} />
          )}
        </IconWrapper>
      )}

      {(!!title || !!subtitle) && (
        <TextWrapper>
          {!!title && (
            <Title size={size} family={family} $isPale={isPale} $status={statusType}>
              {title}
            </Title>
          )}

          {!!subtitle && (
            <TextWrapper>
              <Divider orientation='vertical' length={`${size - 2}px`} type={isPale ? undefined : statusType} />
              <SubTitle size={size - 2} family={family} $isPale={isPale} $status={statusType}>
                {subtitle}
              </SubTitle>
            </TextWrapper>
          )}
        </TextWrapper>
      )}
    </StatusWrapper>
  );
};
