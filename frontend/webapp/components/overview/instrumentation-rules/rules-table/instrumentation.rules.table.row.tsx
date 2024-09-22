import React from 'react';
import theme from '@/styles/palette';
import { KeyvalText } from '@/design.system';
import styled, { css } from 'styled-components';
import { INSTRUMENTATION_RULES } from '@/utils';
import { PayloadCollectionIcon } from '@/assets';
import { InstrumentationRuleSpec } from '@/types';
import RuleRowDynamicContent from './rule.row.dynamic.content';

const StyledTr = styled.tr`
  &:hover {
    background-color: ${theme.colors.light_dark};
  }
`;

const StyledTd = styled.td<{ isFirstRow?: boolean }>`
  padding: 10px 20px;
  border-top: 1px solid ${theme.colors.blue_grey};
  display: flex;
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

const RuleIconContainer = styled.div`
  display: flex;
  gap: 8px;
  margin-left: 10px;
  width: 100%;
`;

const RuleDetails = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

const ClusterAttributesContainer = styled.div`
  display: flex;
  gap: 8px;
  align-items: center;
`;

export function InstrumentationRulesTableRow({
  item,
  index,
  onRowClick,
}: {
  item: InstrumentationRuleSpec;
  index: number;
  onRowClick: (id: string) => void;
}) {
  return (
    <StyledTr key={item?.ruleId}>
      <StyledMainTd
        isFirstRow={index === 0}
        onClick={() => onRowClick(item?.ruleId || '')}
      >
        <RuleIconContainer>
          <div style={{ height: 16 }}>
            <PayloadCollectionIcon style={{ width: 16, height: 16 }} />
          </div>
          <RuleDetails>
            <KeyvalText color={theme.colors.light_grey} size={12}>
              {INSTRUMENTATION_RULES['payload-collection'].TITLE}
            </KeyvalText>
            <ClusterAttributesContainer>
              <KeyvalText data-cy={'rules-rule-name'} weight={600}>
                {`${item?.ruleName || 'Rule'}`}
              </KeyvalText>
              <RuleRowDynamicContent item={item} />
            </ClusterAttributesContainer>
            <KeyvalText color={theme.text.light_grey} size={14}>
              {item?.notes}
            </KeyvalText>
          </RuleDetails>
        </RuleIconContainer>
      </StyledMainTd>
    </StyledTr>
  );
}
