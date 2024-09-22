import React, { useMemo } from 'react';
import theme from '@/styles/palette';
import { InstrumentationRuleSpec, RuleData } from '@/types';
import styled, { css } from 'styled-components';
import { KeyvalCheckbox, KeyvalText } from '@/design.system';
import RuleRowDynamicContent from './rule.row.dynamic.content';
import { INSTRUMENTATION_RULES } from '@/utils';
import { PayloadCollectionIcon } from '@/assets';

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

const StatusIndicator = styled.div<{ disabled: boolean }>`
  width: 6px;
  height: 6px;
  border-radius: 4px;
  background-color: ${({ disabled }) =>
    disabled ? theme.colors.orange_brown : theme.colors.success};
`;

export function InstrumentationRulesTableRow({
  item,
  index,
  onRowClick,
}: {
  item: InstrumentationRuleSpec;
  index: number;
  data: InstrumentationRuleSpec[];
  selectedCheckbox: string[];
  onSelectedCheckboxChange: (id: string) => void;
  onRowClick: (id: string) => void;
}) {
  return (
    <StyledTr key={'TODO'}>
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
                {`${item.ruleName || 'Rule'}`}
              </KeyvalText>
              <RuleRowDynamicContent item={item} />
            </ClusterAttributesContainer>
            <KeyvalText color={theme.text.light_grey} size={14}>
              {item.notes}
            </KeyvalText>
          </RuleDetails>
        </RuleIconContainer>
      </StyledMainTd>
    </StyledTr>
  );
}
