import React, { Fragment, useCallback, useEffect, useRef, useState } from 'react';
import { NOTIFICATION_TYPE } from '@/types';
import styled, { css } from 'styled-components';
import { Divider, ExtendArrow, FlexColumn, FlexRow, IconButton, IconWrapped, MonitorsIcons, Status, Text, Theme, Tooltip, Types } from '@odigos/ui-components';

interface Props {
  title: string;
  subTitle?: string;
  icon?: Types.SVG;
  iconSrc?: string;
  hoverText?: string;
  monitors?: Types.SIGNAL_TYPE[];
  monitorsWithLabels?: boolean;
  isActive?: boolean;
  isError?: boolean;
  withExtend?: boolean;
  isExtended?: boolean;
  renderExtended?: () => React.ReactNode;
  renderActions?: () => React.ReactNode;
  onClick?: () => void;
}

const MAX_TITLE_WIDTH = 160;

const ControlledVisibility = styled.div`
  visibility: hidden;
`;

const Container = styled.div<{ $withClick: boolean; $isError: Props['isError'] }>`
  display: flex;
  flex-direction: column;
  align-self: stretch;
  padding: 16px;
  width: calc(100% - 32px);
  border-radius: 16px;
  background-color: ${({ $isError, theme }) => ($isError ? theme.text.error + Theme.hexPercent['010'] : theme.colors.secondary + Theme.hexPercent['005'])};

  ${({ $withClick, $isError, theme }) =>
    $withClick &&
    css`
      &:hover {
        cursor: pointer;
        background-color: ${$isError ? theme.text.error + Theme.hexPercent['020'] : theme.colors.secondary + Theme.hexPercent['010']};
        ${ControlledVisibility} {
          visibility: visible;
        }
      }
    `}

  &:hover {
    ${ControlledVisibility} {
      visibility: visible;
    }
  }
`;

const Title = styled(Text)`
  max-width: ${MAX_TITLE_WIDTH}px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-size: 14px;
  &::after {
    // This is to prevent the browser "default tooltip" from appearing when the title is too long
    content: '';
    display: block;
  }
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

const HoverText = styled(Text)`
  margin-right: 16px;
`;

export const DataTab: React.FC<Props> = ({
  title,
  subTitle,
  icon,
  iconSrc,
  hoverText,
  monitors,
  monitorsWithLabels,
  isActive,
  isError,
  withExtend,
  isExtended,
  renderExtended,
  renderActions,
  onClick,
  ...props
}) => {
  const [extend, setExtend] = useState(isExtended || false);
  const [isTitleOverflowed, setIsTitleOverflowed] = useState(false);
  const titleRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const { current } = titleRef;

    if (current) {
      const { clientWidth } = current;
      const marginUp = MAX_TITLE_WIDTH * 1.05; // add 5%
      const marginDown = MAX_TITLE_WIDTH * 0.95; // subtract 5%

      setIsTitleOverflowed(clientWidth < marginUp && clientWidth > marginDown);
    }
  }, [title]);

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
          <Status status={isActive ? NOTIFICATION_TYPE.SUCCESS : NOTIFICATION_TYPE.ERROR} size={10} />
        </>
      );
    },
    [isActive],
  );

  return (
    <Container $isError={isError} $withClick={!!onClick} onClick={onClick} {...props}>
      <FlexRow $gap={8}>
        <IconWrapped icon={icon} src={iconSrc} status={isError ? NOTIFICATION_TYPE.ERROR : undefined} />

        <FlexColumn $gap={4}>
          {isTitleOverflowed ? (
            <Tooltip text={title} withIcon={false}>
              <Title ref={titleRef}>{title}</Title>
            </Tooltip>
          ) : (
            <Title ref={titleRef}>{title}</Title>
          )}
          <SubTitleWrapper>
            {subTitle && <SubTitle>{subTitle}</SubTitle>}
            {renderMonitors(!!subTitle)}
            {renderActiveStatus(!!monitors?.length)}
          </SubTitleWrapper>
        </FlexColumn>

        <ActionsWrapper>
          {!!hoverText && (
            <ControlledVisibility>
              <HoverText size={14} family='secondary'>
                {hoverText}
              </HoverText>
            </ControlledVisibility>
          )}
          {renderActions && renderActions()}
          {withExtend && (
            <Fragment>
              <Divider orientation='vertical' length='16px' margin='0 2px' />
              <IconButton onClick={() => setExtend((prev) => !prev)}>
                <ExtendArrow extend={extend} />
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
