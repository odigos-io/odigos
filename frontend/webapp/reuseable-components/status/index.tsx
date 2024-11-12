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

const StatusWrapper = styled.div<Props>`
  display: flex;
  align-items: center;
  width: fit-content;
  padding: ${({ withIcon, withBorder, withSmaller }) => (withIcon || withBorder ? (withSmaller ? '4px 8px' : '8px 24px') : '0')};
  border-radius: 32px;
  border: 1px solid
    ${({ withBorder, isActive, theme }) => (withBorder ? (isActive ? theme.colors.dark_green : theme.colors.dark_red) : 'transparent')};
  background: ${({ withBackground, isActive }) =>
    withBackground
      ? isActive
        ? `linear-gradient(90deg, rgba(23, 32, 19, 0) 0%, rgba(23, 32, 19, 0.8) 50%, #172013 100%)`
        : `linear-gradient(90deg, rgba(51, 21, 21, 0.00) 0%, rgba(51, 21, 21, 0.80) 50%, #331515 100%)`
      : 'transparent'};
`;

const IconWrapper = styled.div<Props>`
  display: flex;
  align-items: center;
  margin-right: ${({ withSmaller }) => (withSmaller ? '6px' : '8px')};
`;

const TextWrapper = styled.div<Props>`
  display: flex;
  align-items: center;
`;

const Title = styled(Text)<Props>`
  font-weight: 400;
  font-size: ${({ withSmaller }) => (withSmaller ? '12px' : '14px')};
  font-family: ${({ withSpecialFont, theme }) => (withSpecialFont ? theme.font_family.secondary : theme.font_family.primary)};
  color: ${({ isActive, theme }) => (isActive ? theme.text.success : theme.text.error)};
  text-transform: ${({ withSpecialFont }) => (withSpecialFont ? 'uppercase' : 'unset')};
`;

const SubTitle = styled(Text)<Props>`
  font-weight: 400;
  font-size: ${({ withSmaller }) => (withSmaller ? '10px' : '12px')};
  font-family: ${({ withSpecialFont, theme }) => (withSpecialFont ? theme.font_family.secondary : theme.font_family.primary)};
  color: ${({ isActive, theme }) => (isActive ? theme.colors.green['600'] : theme.colors.red['600'])};
  text-transform: ${({ withSpecialFont }) => (withSpecialFont ? 'uppercase' : 'unset')};
`;

const TextDivider = styled.div<Props>`
  width: 1px;
  height: 12px;
  background: ${({ isActive }) => (isActive ? 'rgba(124, 237, 124, 0.16)' : 'rgba(237, 124, 124, 0.16)')};
  margin: 0 8px;
`;

const Status: React.FC<Props> = (props) => {
  const { title, subtitle, isActive, withIcon, withSmaller } = props;

  return (
    <StatusWrapper {...props}>
      {withIcon && (
        <IconWrapper {...props}>
          <Image src={getStatusIcon(isActive ? 'success' : 'error')} alt='status' width={withSmaller ? 14 : 16} height={withSmaller ? 14 : 16} />
        </IconWrapper>
      )}

      <TextWrapper {...props}>
        <Title {...props}>{title || (isActive ? 'Active' : 'Inactive')}</Title>

        {subtitle && (
          <TextWrapper {...props}>
            <TextDivider {...props} />
            <SubTitle {...props}>{subtitle}</SubTitle>
          </TextWrapper>
        )}
      </TextWrapper>
    </StatusWrapper>
  );
};

export { Status };
