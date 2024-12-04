import React, { PropsWithChildren, useCallback } from 'react';
import Image from 'next/image';
import { FlexColumn } from '@/styles';
import styled, { css } from 'styled-components';
import { ActiveStatus, MonitorsIcons, Text } from '@/reuseable-components';

interface Props extends PropsWithChildren {
  title: string;
  subTitle: string;
  logo: string;
  monitors?: string[];
  isActive?: boolean;
  isError?: boolean;
  onClick?: () => void;
}

const Container = styled.div<{ $withClick: boolean; $isError: Props['isError'] }>`
  display: flex;
  align-items: center;
  align-self: stretch;
  gap: 8px;
  padding: 16px;
  width: calc(100% - 32px);
  border-radius: 16px;
  background-color: ${({ $isError, theme }) => ($isError ? '#281515' : theme.colors.white_opacity['004'])};

  ${({ $withClick, $isError, theme }) =>
    $withClick &&
    css`
      &:hover {
        cursor: pointer;
        background-color: ${$isError ? '#351515' : theme.colors.white_opacity['008']};
      }
    `}
`;

const IconWrapper = styled.div<{ $isError: Props['isError'] }>`
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: ${({ $isError }) =>
    `linear-gradient(180deg, ${$isError ? 'rgba(237, 124, 124, 0.08)' : 'rgba(249, 249, 249, 0.06)'} 0%, ${$isError ? 'rgba(237, 124, 124, 0.02)' : 'rgba(249, 249, 249, 0.02)'} 100%)`};
`;

const Title = styled(Text)`
  max-width: 150px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
`;

const SubTitleWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;

const SubTitle = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
  font-size: 10px;
`;

const ActionsWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
`;

export const DataTab: React.FC<Props> = ({ title, subTitle, logo, monitors, isActive, isError, onClick, children }) => {
  const renderMonitors = useCallback(() => {
    if (!monitors) return null;

    return (
      <>
        <SubTitle>{'•'}</SubTitle>
        <MonitorsIcons monitors={monitors} size={10} />
      </>
    );
  }, [monitors]);

  const renderActiveStatus = useCallback(() => {
    if (typeof isActive !== 'boolean') return null;

    return (
      <>
        <SubTitle>{'•'}</SubTitle>
        <ActiveStatus isActive={isActive} size={10} />
      </>
    );
  }, [isActive]);

  return (
    <Container $isError={isError} $withClick={!!onClick} onClick={onClick}>
      <IconWrapper $isError={isError}>
        <Image src={logo} alt='' width={20} height={20} />
      </IconWrapper>

      <FlexColumn>
        <Title>{title}</Title>
        <SubTitleWrapper>
          <SubTitle>{subTitle}</SubTitle>
          {renderMonitors()}
          {renderActiveStatus()}
        </SubTitleWrapper>
      </FlexColumn>

      <ActionsWrapper>{children}</ActionsWrapper>
    </Container>
  );
};
