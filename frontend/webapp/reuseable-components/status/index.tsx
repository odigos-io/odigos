import React from 'react';
import Image from 'next/image';
import styled, { css } from 'styled-components';
import { getStatusIcon } from '@/utils';
import { Divider, Text } from '@/reuseable-components';
import theme, { hexPercentValues } from '@/styles/theme';

export * from './active-status';
export * from './connection-status';
export * from './instrument-status';

export interface StatusProps {
  title?: string;
  subtitle?: string;
  size?: number;
  family?: 'primary' | 'secondary';
  isPale?: boolean;
  isActive?: boolean;
  withIcon?: boolean;
  withBorder?: boolean;
  withBackground?: boolean;
}

const StatusWrapper = styled.div<{
  $size: number;
  $isPale: StatusProps['isPale'];
  $isActive: StatusProps['isActive'];
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
  border: ${({ $withBorder, $isPale, $isActive, theme }) => ($withBorder ? `1px solid ${$isPale ? theme.colors.border : $isActive ? theme.colors.dark_green : theme.colors.dark_red}` : 'none')};
  background: ${({ $withBackground, $isPale, $isActive, theme }) =>
    $withBackground
      ? $isPale
        ? `linear-gradient(90deg, transparent 0%, ${theme.colors.info + hexPercentValues['080']} 50%, ${theme.colors.info} 100%)`
        : $isActive
        ? `linear-gradient(90deg, transparent 0%, ${theme.colors.success + hexPercentValues['080']} 50%, ${theme.colors.success} 100%)`
        : `linear-gradient(90deg, transparent 0%, ${theme.colors.error + hexPercentValues['080']} 50%, ${theme.colors.error} 100%)`
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
  $isActive: StatusProps['isActive'];
}>`
  color: ${({ $isPale, $isActive, theme }) => ($isPale ? theme.text.secondary : $isActive ? theme.text.success : theme.text.error)};
`;

const SubTitle = styled(Text)<{
  $isPale: StatusProps['isPale'];
  $isActive: StatusProps['isActive'];
}>`
  color: ${({ $isPale, $isActive }) => ($isPale ? theme.text.grey : $isActive ? '#51DB51' : '#DB5151')};
`;

export const Status: React.FC<StatusProps> = ({ title, subtitle, size = 12, family = 'secondary', isPale, isActive, withIcon, withBorder, withBackground }) => {
  return (
    <StatusWrapper $size={size} $isPale={isPale} $isActive={isActive} $withIcon={withIcon} $withBorder={withBorder} $withBackground={withBackground}>
      {withIcon && (
        <IconWrapper>
          {/* TODO: SVG to JSX */}
          <Image src={isPale ? `/icons/common/circled-${isActive ? 'check' : 'cross'}.svg` : getStatusIcon(isActive ? 'success' : 'error')} alt='status' width={size + 2} height={size + 2} />
        </IconWrapper>
      )}

      {(!!title || !!subtitle) && (
        <TextWrapper>
          {!!title && (
            <Title size={size} family={family} $isPale={isPale} $isActive={isActive}>
              {title}
            </Title>
          )}

          {!!subtitle && (
            <TextWrapper>
              <Divider orientation='vertical' length={`${size - 2}px`} type={isPale ? undefined : isActive ? 'success' : 'error'} />
              <SubTitle size={size - 2} family={family} $isPale={isPale} $isActive={isActive}>
                {subtitle}
              </SubTitle>
            </TextWrapper>
          )}
        </TextWrapper>
      )}
    </StatusWrapper>
  );
};
