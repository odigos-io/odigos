import React, { useMemo } from 'react';
import theme from '@/styles/palette';
import { Destination } from '@/types';
import styled, { css } from 'styled-components';
import { KeyvalImage, KeyvalText } from '@/design.system';
import { MONITORING_OPTIONS } from '@/utils/constants/monitors';
import { TapList } from '@/components/setup/destination/tap.list/tap.list';
import { ConditionCheck } from '@/components/common';

const StyledTr = styled.tr`
  &:hover {
    background-color: ${theme.colors.light_dark};
  }
`;

const StyledTd = styled.td<{ isFirstRow?: boolean }>`
  padding: 10px 20px;
  border-top: 1px solid ${theme.colors.blue_grey};

  ${({ isFirstRow }) =>
    isFirstRow &&
    css`
      border-top: none;
    `}
`;

const StyledMainTd = styled(StyledTd)`
  cursor: pointer;
  padding: 10px 20px;
`;

const SourceIconContainer = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;

const SourceDetails = styled.div`
  display: flex;
  flex-direction: column;
`;

const NameContainer = styled.div`
  display: flex;
  gap: 10px;
  align-items: center;
`;

const FooterContainer = styled.div`
  display: flex;
  align-items: center;
`;

const FooterItemWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 4px;
`;

const StatusIndicator = styled.div<{ color: string }>`
  width: 6px;
  height: 6px;
  border-radius: 4px;
  background-color: ${({ color }) => color};
`;

const TagWrapper = styled.div`
  padding: 0 20px;
  width: 300px;
  display: flex;
  align-items: center;
`;

const LOGO_STYLE: React.CSSProperties = {
  padding: 4,
  backgroundColor: theme.colors.white,
};
const TAP_STYLE: React.CSSProperties = { padding: '4px 8px', gap: 4 };
export function DestinationsTableRow({
  item,
  index,
  onRowClick,
}: {
  item: Destination;
  index: number;
  onRowClick: (source: Destination) => void;
}) {
  const monitors = useMemo(() => {
    const supported_signals = item.destination_type.supported_signals;
    const signals = item.signals;
    return Object?.entries(supported_signals).reduce((acc, [key, _]) => {
      const monitor = MONITORING_OPTIONS.find(
        (option) => option.title.toLowerCase() === key
      );
      if (monitor && supported_signals[key].supported) {
        return [...acc, { ...monitor, tapped: signals[key] }];
      }

      return acc;
    }, []);
  }, [JSON.stringify(item.signals)]);

  return (
    <StyledTr key={item.id}>
      <StyledMainTd onClick={() => onRowClick(item)} isFirstRow={index === 0}>
        <SourceIconContainer>
          <div>
            <KeyvalImage
              src={item.destination_type.image_url || ''}
              width={32}
              height={32}
              style={LOGO_STYLE}
              alt="source-logo"
            />
          </div>
          <SourceDetails>
            <NameContainer>
              <KeyvalText color={theme.text.light_grey} size={14}>
                {item.destination_type.display_name}
              </KeyvalText>
              <ConditionCheck conditions={item.conditions} />
            </NameContainer>
            <FooterContainer>
              <KeyvalText size={20} weight={600}>
                {`${item.name || 'Destination'} `}
              </KeyvalText>
            </FooterContainer>
          </SourceDetails>

          <TagWrapper>
            <TapList gap={4} list={monitors} tapStyle={TAP_STYLE} />
          </TagWrapper>
        </SourceIconContainer>
      </StyledMainTd>
    </StyledTr>
  );
}
