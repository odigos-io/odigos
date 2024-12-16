import React, { Fragment, useCallback, useState } from 'react';
import Image from 'next/image';
import { FlexColumn, FlexRow } from '@/styles';
import styled, { css } from 'styled-components';
import { ActiveStatus, Divider, ExtendIcon, IconButton, MonitorsIcons, Text } from '@/reuseable-components';

interface Props {
  title: string;
  subTitle: string;
  logo: string;
  monitors?: string[];
  monitorsWithLabels?: boolean;
  isActive?: boolean;
  isError?: boolean;
  withExtend?: boolean;
  isExtended?: boolean;
  renderExtended?: () => JSX.Element;
  renderActions?: () => JSX.Element;
  onClick?: () => void;
}

const Container = styled.div<{ $withClick: boolean; $isError: Props['isError'] }>`
  display: flex;
  flex-direction: column;
  align-self: stretch;
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
  font-size: 14px;
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

export const DataTab: React.FC<Props> = ({ title, subTitle, logo, monitors, monitorsWithLabels, isActive, isError, withExtend, isExtended, renderExtended, renderActions, onClick }) => {
  const [extend, setExtend] = useState(isExtended || false);

  const renderMonitors = useCallback(
    (withSeperator: boolean) => {
      if (!monitors || !monitors.length) return null;

      return (
        <>
          {withSeperator && <SubTitle>{'•'}</SubTitle>}
          <MonitorsIcons monitors={monitors} withLabels={monitorsWithLabels} size={10} />
        </>
      );
    },
    [monitors],
  );

  const renderActiveStatus = useCallback(
    (withSeperator: boolean) => {
      if (typeof isActive !== 'boolean') return null;

      return (
        <>
          {withSeperator && <SubTitle>{'•'}</SubTitle>}
          <ActiveStatus isActive={isActive} size={10} />
        </>
      );
    },
    [isActive],
  );

  return (
    <Container $isError={isError} $withClick={!!onClick} onClick={onClick}>
      <FlexRow $gap={8}>
        <IconWrapper $isError={isError}>
          <Image src={logo} alt='' width={20} height={20} />
        </IconWrapper>

        <FlexColumn $gap={4}>
          <Title>{title}</Title>
          <SubTitleWrapper>
            {subTitle && <SubTitle>{subTitle}</SubTitle>}
            {renderMonitors(!!subTitle)}
            {renderActiveStatus(!!monitors?.length)}
          </SubTitleWrapper>
        </FlexColumn>

        <ActionsWrapper>
          {renderActions && renderActions()}
          {withExtend && (
            <Fragment>
              <Divider orientation='vertical' length='16px' margin='0 2px' />
              <IconButton onClick={() => setExtend((prev) => !prev)}>
                <ExtendIcon extend={extend} />
              </IconButton>
            </Fragment>
          )}
        </ActionsWrapper>
      </FlexRow>

      {extend && renderExtended && (
        <FlexColumn>
          <Divider margin='16px 0' />
          {renderExtended()}
        </FlexColumn>
      )}
    </Container>
  );
};
