import React from 'react';
import theme from '@/styles/palette';
import { ManagedSource } from '@/types';
import { Container, Namespace } from '@/assets';
import styled, { css } from 'styled-components';
import {
  KeyvalCheckbox,
  KeyvalImage,
  KeyvalLoader,
  KeyvalTag,
  KeyvalText,
} from '@/design.system';
import { KIND_COLORS } from '@/styles/global';
import {
  LANGUAGES_COLORS,
  LANGUAGES_LOGOS,
  getMainContainerLanguage,
} from '@/utils';
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
  display: flex;
  gap: 20px;
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

function getFirstNonIgnoredContainerName(
  managedSource: ManagedSource
): string | null {
  if (!managedSource?.instrumented_application_details?.languages) {
    return null;
  }

  const nonIgnoredLanguage =
    managedSource?.instrumented_application_details?.languages.find(
      (language) => language.language !== 'ignore'
    );

  return nonIgnoredLanguage ? nonIgnoredLanguage.container_name : null;
}

const DEPLOYMENT = 'deployment';
export function SourcesTableRow({
  item,
  index,
  selectedCheckbox,
  onSelectedCheckboxChange,
  onRowClick,
}: {
  item: ManagedSource;
  index: number;
  data: ManagedSource[];
  selectedCheckbox: string[];
  onSelectedCheckboxChange: (id: string) => void;
  onRowClick: (source: ManagedSource) => void;
}) {
  const workloadProgrammingLanguage = getMainContainerLanguage(item);

  const containerName = getFirstNonIgnoredContainerName(item) || '';

  function getLanguageStatus() {
    if (workloadProgrammingLanguage === 'processing') {
      return (
        <>
          <KeyvalLoader width={6} height={6} />
          <KeyvalText color={theme.text.light_grey} size={14}>
            {'detecting language'}
          </KeyvalText>
        </>
      );
    }

    return (
      <>
        <StatusIndicator
          color={
            LANGUAGES_COLORS[workloadProgrammingLanguage] ||
            theme.text.light_grey
          }
        />
        <KeyvalText color={theme.text.light_grey} size={14}>
          {workloadProgrammingLanguage}
        </KeyvalText>
      </>
    );
  }

  return (
    <StyledTr key={item.kind}>
      <StyledMainTd isFirstRow={index === 0}>
        <KeyvalCheckbox
          value={selectedCheckbox.includes(JSON.stringify(item))}
          onChange={() => onSelectedCheckboxChange(JSON.stringify(item))}
        />
        <SourceIconContainer onClick={() => onRowClick(item)}>
          <div>
            <KeyvalImage
              src={LANGUAGES_LOGOS[workloadProgrammingLanguage]}
              width={32}
              height={32}
              style={LOGO_STYLE}
              alt="source-logo"
            />
          </div>
          <SourceDetails onClick={() => onRowClick(item)}>
            <NameContainer>
              <KeyvalText weight={600}>
                {`${item.name || 'Source'} `}
              </KeyvalText>
              <KeyvalText color={theme.text.light_grey} size={14}>
                <ConditionCheck
                  conditions={
                    item?.instrumented_application_details?.conditions || []
                  }
                />
              </KeyvalText>
            </NameContainer>
            <FooterContainer>
              <FooterItemWrapper>{getLanguageStatus()}</FooterItemWrapper>
              <FooterItemWrapper>
                <Namespace style={{ width: 16, height: 16 }} />
                <KeyvalText color={theme.text.light_grey} size={14}>
                  {item.namespace}
                </KeyvalText>
              </FooterItemWrapper>
              <FooterItemWrapper>
                <Container style={{ width: 16, height: 16 }} />
                <KeyvalText color={theme.text.light_grey} size={14}>
                  {containerName}
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
