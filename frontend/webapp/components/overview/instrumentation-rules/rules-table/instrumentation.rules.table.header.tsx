import React, { useState } from 'react';
import { OVERVIEW } from '@/utils';
import theme from '@/styles/palette';
import styled from 'styled-components';
import { RulesSortType } from '@/types';
import { ActionsGroup, KeyvalText } from '@/design.system';

const StyledThead = styled.div`
  background-color: ${theme.colors.light_dark};
  border-top-right-radius: 6px;
  border-top-left-radius: 6px;
`;

const StyledTh = styled.th`
  padding: 10px 20px;
  text-align: left;
  border-bottom: 1px solid ${theme.colors.blue_grey};
`;

const StyledMainTh = styled(StyledTh)`
  padding: 10px 20px;
  display: flex;
  align-items: center;
  gap: 8px;
`;

const ActionGroupContainer = styled.div`
  display: flex;
  justify-content: flex-end;
  padding-right: 20px;
  gap: 24px;
  flex: 1;
`;

interface InstrumentationRulesTableHeaderProps {
  data: any[];
  sortRules?: (condition: string) => void;
}

export function InstrumentationRulesTableHeader({
  data,
  sortRules,
}: InstrumentationRulesTableHeaderProps) {
  const [currentSortId, setCurrentSortId] = useState('');

  function onSortClick(id: string) {
    setCurrentSortId(id);
    sortRules && sortRules(id);
  }

  const ruleGroups = [
    {
      label: 'Sort by',
      subTitle: 'Sort by criteria',
      items: [
        {
          label: 'Type',
          onClick: () => onSortClick(RulesSortType.TYPE),
          id: RulesSortType.TYPE,
          selected: currentSortId === RulesSortType.TYPE,
        },
        {
          label: 'Rule Name',
          onClick: () => onSortClick(RulesSortType.RULE_NAME),
          id: RulesSortType.RULE_NAME,
          selected: currentSortId === RulesSortType.RULE_NAME,
        },
        {
          label: 'Status',
          onClick: () => onSortClick(RulesSortType.STATUS),
          id: RulesSortType.STATUS,
          selected: currentSortId === RulesSortType.STATUS,
        },
      ],
      condition: true,
    },
  ];

  return (
    <StyledThead>
      <StyledMainTh>
        <KeyvalText size={14} weight={600} color={theme.text.white}>
          {`${data.length} ${OVERVIEW.MENU.INSTRUMENTATION_RULES}`}
        </KeyvalText>
        <ActionGroupContainer>
          <ActionsGroup actionGroups={ruleGroups} />
        </ActionGroupContainer>
      </StyledMainTh>
    </StyledThead>
  );
}
