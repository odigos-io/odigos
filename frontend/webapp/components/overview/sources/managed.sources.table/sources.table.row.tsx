import React from 'react';
import theme from '@/styles/palette';
import { ManagedSource } from '@/types';
import { Container, Namespace } from '@/assets';
import styled, { css } from 'styled-components';
import {
  KeyvalCheckbox,
  KeyvalImage,
  KeyvalTag,
  KeyvalText,
} from '@/design.system';
import { LANGUAGES_LOGOS } from '@/assets/images';
import { KIND_COLORS } from '@/styles/global';
import { LANGUAGES_COLORS } from '@/assets/images/sources.card/sources.card';

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
  gap: 4px;
`;

const NameContainer = styled.div`
  display: flex;
  gap: 10px;
  align-items: center;
`;

const FooterContainer = styled.div`
  display: flex;
  gap: 16px;
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

const DEPLOYMENT = 'deployment';
export function SourcesTableRow({
  item,
  index,

  onRowClick,
}: {
  item: ManagedSource;
  index: number;
  data: ManagedSource[];

  onRowClick: (source: ManagedSource) => void;
}) {
  return (
    <StyledTr key={item.kind}>
      <StyledMainTd onClick={() => onRowClick(item)} isFirstRow={index === 0}>
        <SourceIconContainer>
          <div>
            <KeyvalImage
              src={LANGUAGES_LOGOS[item?.languages?.[0].language || '']}
              width={32}
              height={32}
              style={LOGO_STYLE}
              alt="source-logo"
            />
          </div>
          <SourceDetails>
            <NameContainer>
              <KeyvalText weight={600}>
                {`${item.name || 'Source'} `}
              </KeyvalText>
              <KeyvalText weight={600}>
                {`${item.reported_name || ''} `}
              </KeyvalText>
            </NameContainer>
            <FooterContainer>
              <FooterItemWrapper>
                <StatusIndicator
                  color={
                    LANGUAGES_COLORS[item?.languages?.[0].language] ||
                    theme.text.light_grey
                  }
                />
                <KeyvalText color={theme.text.light_grey} size={14}>
                  {item?.languages?.[0].language}
                </KeyvalText>
              </FooterItemWrapper>
              <FooterItemWrapper>
                <Namespace
                  style={{
                    width: 16,
                    height: 16,
                  }}
                />
                <KeyvalText color={theme.text.light_grey} size={14}>
                  {item.namespace}
                </KeyvalText>
              </FooterItemWrapper>
              <FooterItemWrapper>
                <Container
                  style={{
                    width: 16,
                    height: 16,
                  }}
                />
                <KeyvalText color={theme.text.light_grey} size={14}>
                  {item?.languages?.[0].container_name}
                </KeyvalText>
              </FooterItemWrapper>
            </FooterContainer>
          </SourceDetails>

          <TagWrapper>
            <KeyvalTag
              title={item?.kind || ''}
              color={KIND_COLORS[item?.kind?.toLowerCase() || DEPLOYMENT]}
            />
          </TagWrapper>
        </SourceIconContainer>
      </StyledMainTd>
    </StyledTr>
  );
}
