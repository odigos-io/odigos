import React, { useState } from 'react';
import { ACTION, MONITORS, OVERVIEW } from '@/utils';
import theme from '@/styles/palette';
import styled from 'styled-components';
import { ActionsSortType, SourceSortOptions } from '@/types';
import { UnFocusAction, UnFocusSources } from '@/assets/icons/side.menu';
import { ActionsGroup, KeyvalCheckbox, KeyvalText } from '@/design.system';

const StyledThead = styled.thead`
  background-color: ${theme.colors.light_dark};
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

const SELECT_ALL_CHECKBOX = 'select_all';

interface ActionsTableHeaderProps {
  data: any[];
  sortSources?: (condition: string) => void;
  filterActionsBySignal?: (signals: string[]) => void;
  toggleActionStatus?: (ids: string[], disabled: boolean) => void;
}

export function SourcesTableHeader({
  data,

  sortSources,
  filterActionsBySignal,
  toggleActionStatus,
}: ActionsTableHeaderProps) {
  const [currentSortId, setCurrentSortId] = useState('');
  const [groupActions, setGroupActions] = useState([
    'traces',
    'logs',
    'metrics',
  ]);
  function onSortClick(id: string) {
    setCurrentSortId(id);
    sortSources && sortSources(id);
  }

  function onGroupClick(id: string) {
    let newGroup: string[] = [];
    if (groupActions.includes(id)) {
      setGroupActions(groupActions.filter((item) => item !== id));
      newGroup = groupActions.filter((item) => item !== id);
    } else {
      setGroupActions([...groupActions, id]);
      newGroup = [...groupActions, id];
    }

    filterActionsBySignal && filterActionsBySignal(newGroup);
  }

  const actionGroups = [
    // {
    //   label: 'Metrics',
    //   subTitle: 'Display',
    //   items: [
    //     {
    //       label: MONITORS.TRACES,
    //       onClick: () => onGroupClick(MONITORS.TRACES.toLowerCase()),
    //       id: MONITORS.TRACES.toLowerCase(),
    //       selected: groupActions.includes(MONITORS.TRACES.toLowerCase()),
    //       disabled:
    //         groupActions.length === 1 &&
    //         groupActions.includes(MONITORS.TRACES.toLowerCase()),
    //     },
    //     {
    //       label: MONITORS.LOGS,
    //       onClick: () => onGroupClick(MONITORS.LOGS.toLowerCase()),
    //       id: MONITORS.LOGS.toLowerCase(),
    //       selected: groupActions.includes(MONITORS.LOGS.toLowerCase()),
    //       disabled:
    //         groupActions.length === 1 &&
    //         groupActions.includes(MONITORS.LOGS.toLowerCase()),
    //     },
    //     {
    //       label: MONITORS.METRICS,
    //       onClick: () => onGroupClick(MONITORS.METRICS.toLowerCase()),
    //       id: MONITORS.METRICS.toLowerCase(),
    //       selected: groupActions.includes(MONITORS.METRICS.toLowerCase()),
    //       disabled:
    //         groupActions.length === 1 &&
    //         groupActions.includes(MONITORS.METRICS.toLowerCase()),
    //     },
    //   ],
    //   condition: true,
    // },
    {
      label: 'Sort by',
      subTitle: 'Sort by',
      items: [
        {
          label: 'Kind',
          onClick: () => onSortClick(SourceSortOptions.KIND),
          id: SourceSortOptions.KIND,
          selected: currentSortId === SourceSortOptions.KIND,
        },
        {
          label: 'Language',
          onClick: () => onSortClick(SourceSortOptions.LANGUAGE),
          id: SourceSortOptions.LANGUAGE,
          selected: currentSortId === SourceSortOptions.LANGUAGE,
        },
        {
          label: 'Name',
          onClick: () => onSortClick(SourceSortOptions.NAME),
          id: SourceSortOptions.NAME,
          selected: currentSortId === SourceSortOptions.NAME,
        },
        {
          label: 'Namespace',
          onClick: () => onSortClick(SourceSortOptions.NAMESPACE),
          id: SourceSortOptions.NAMESPACE,
          selected: currentSortId === SourceSortOptions.NAMESPACE,
        },
      ],
      condition: true,
    },
  ];

  return (
    <StyledThead>
      <StyledMainTh>
        <UnFocusSources style={{ width: 18, height: 18 }} />
        <KeyvalText size={14} weight={600} color={theme.text.white}>
          {`${data.length} ${OVERVIEW.MENU.ACTIONS}`}
        </KeyvalText>
        <ActionGroupContainer>
          <ActionsGroup actionGroups={actionGroups} />
        </ActionGroupContainer>
      </StyledMainTh>
    </StyledThead>
  );
}
