import React from 'react';
import Image from 'next/image';
import { Text } from '../text';
import styled from 'styled-components';
import { getStatusIcon } from '@/utils';

interface Props {
  title?: string;
  subtitle?: string;
  isActive?: boolean;
  withBackground?: boolean;
  withBorder?: boolean;
  withSmaller?: boolean;
  withSpecialFont?: boolean;
  withIcon?: boolean;
}

const StatusWrapper = styled.div<{
  $isActive?: Props['isActive'];
  $withIcon?: Props['withIcon'];
  $withBorder?: Props['withBorder'];
  $withBackground?: Props['withBackground'];
  $withSmaller?: Props['withSmaller'];
}>`
  display: flex;
  align-items: center;
  width: fit-content;
  padding: ${({ $withIcon, $withBorder, $withSmaller }) => ($withIcon || $withBorder ? ($withSmaller ? '2px 6px' : '8px 24px') : '0')};
  border-radius: 32px;
  border: 1px solid ${({ $withBorder, $isActive, theme }) => ($withBorder ? ($isActive ? theme.colors.dark_green : theme.colors.dark_red) : 'transparent')};
  background: ${({ $withBackground, $isActive }) =>
    $withBackground
      ? $isActive
        ? `linear-gradient(90deg, rgba(23, 32, 19, 0) 0%, rgba(23, 32, 19, 0.8) 50%, #172013 100%)`
        : `linear-gradient(90deg, rgba(51, 21, 21, 0.00) 0%, rgba(51, 21, 21, 0.80) 50%, #331515 100%)`
      : 'transparent'};
`;

const IconWrapper = styled.div<{ $withSmaller?: Props['withSmaller'] }>`
  display: flex;
  align-items: center;
  margin-right: ${({ $withSmaller }) => ($withSmaller ? '6px' : '8px')};
`;

const TextWrapper = styled.div`
  display: flex;
  align-items: center;
`;

const Title = styled(Text)<{ $isActive?: Props['isActive']; $withSpecialFont?: Props['withSpecialFont']; $withSmaller?: Props['withSmaller'] }>`
  font-weight: 400;
  font-size: ${({ $withSmaller }) => ($withSmaller ? '10px' : '14px')};
  font-family: ${({ $withSpecialFont, theme }) => ($withSpecialFont ? theme.font_family.secondary : theme.font_family.primary)};
  color: ${({ $isActive, theme }) => ($isActive ? theme.text.success : theme.text.error)};
  text-transform: ${({ $withSpecialFont }) => ($withSpecialFont ? 'uppercase' : 'unset')};
`;

const SubTitle = styled(Text)<{ $isActive?: Props['isActive']; $withSpecialFont?: Props['withSpecialFont']; $withSmaller?: Props['withSmaller'] }>`
  font-weight: 400;
  font-size: ${({ $withSmaller }) => ($withSmaller ? '8px' : '12px')};
  font-family: ${({ $withSpecialFont, theme }) => ($withSpecialFont ? theme.font_family.secondary : theme.font_family.primary)};
  color: ${({ $isActive }) => ($isActive ? '#51DB51' : '#DB5151')};
  text-transform: ${({ $withSpecialFont }) => ($withSpecialFont ? 'uppercase' : 'unset')};
`;

const TextDivider = styled.div<{ $isActive?: Props['isActive'] }>`
  width: 1px;
  height: 12px;
  background: ${({ $isActive }) => ($isActive ? 'rgba(124, 237, 124, 0.16)' : 'rgba(237, 124, 124, 0.16)')};
  margin: 0 8px;
`;

const Status: React.FC<Props> = ({ title, subtitle, isActive, withIcon, withBorder, withBackground, withSpecialFont, withSmaller }) => {
  return (
    <StatusWrapper $isActive={isActive} $withIcon={withIcon} $withBorder={withBorder} $withBackground={withBackground} $withSmaller={withSmaller}>
      {withIcon && (
        <IconWrapper $withSmaller={withSmaller}>
          <Image src={getStatusIcon(isActive ? 'success' : 'error')} alt='status' width={withSmaller ? 12 : 16} height={withSmaller ? 14 : 16} />
        </IconWrapper>
      )}

      <TextWrapper>
        <Title $isActive={isActive} $withSpecialFont={withSpecialFont} $withSmaller={withSmaller}>
          {title || (isActive ? 'Active' : 'Inactive')}
        </Title>

        {subtitle && (
          <TextWrapper>
            <TextDivider $isActive={isActive} />
            <SubTitle $isActive={isActive} $withSpecialFont={withSpecialFont} $withSmaller={withSmaller}>
              {subtitle}
            </SubTitle>
          </TextWrapper>
        )}
      </TextWrapper>
    </StatusWrapper>
  );
};

export { Status };
