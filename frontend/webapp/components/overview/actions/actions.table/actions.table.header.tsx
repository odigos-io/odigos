import React, { useState } from 'react';
import { MONITORS, OVERVIEW } from '@/utils';
import theme from '@/styles/palette';
import styled from 'styled-components';
import { ActionsSortType } from '@/types';
import { UnFocusAction } from '@/assets/icons/side.menu';
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
  padding: 10px 0px;
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
  selectedCheckbox: string[];
  onSelectedCheckboxChange: (id: string) => void;
  sortActions?: (condition: string) => void;
  filterActionsBySignal?: (signals: string[]) => void;
}

export function ActionsTableHeader({
  data,
  selectedCheckbox,
  onSelectedCheckboxChange,
  sortActions,
  filterActionsBySignal,
}: ActionsTableHeaderProps) {
  const [currentSortId, setCurrentSortId] = useState('');
  const [groupActions, setGroupActions] = useState([
    'traces',
    'logs',
    'metrics',
  ]);
  function onSortClick(id: string) {
    setCurrentSortId(id);
    sortActions && sortActions(id);
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
    {
      label: 'Active Status',
      subTitle: 'Toggle active status',
      disabled: false,
      items: [
        {
          label: 'Enabled',
          onClick: () => console.log('Enabled clicked'),
          id: 'enabled',
        },
        {
          label: 'Disabled',
          onClick: () => console.log('Disabled clicked'),
          id: 'disabled',
        },
      ],
      condition: !!selectedCheckbox.length,
    },

    {
      label: 'Metrics',
      subTitle: 'Display',
      items: [
        {
          label: MONITORS.TRACES,
          onClick: () => onGroupClick(MONITORS.TRACES.toLowerCase()),
          id: MONITORS.TRACES.toLowerCase(),
          selected: groupActions.includes(MONITORS.TRACES.toLowerCase()),
          disabled:
            groupActions.length === 1 &&
            groupActions.includes(MONITORS.TRACES.toLowerCase()),
        },
        {
          label: MONITORS.LOGS,
          onClick: () => onGroupClick(MONITORS.LOGS.toLowerCase()),
          id: MONITORS.LOGS.toLowerCase(),
          selected: groupActions.includes(MONITORS.LOGS.toLowerCase()),
          disabled:
            groupActions.length === 1 &&
            groupActions.includes(MONITORS.LOGS.toLowerCase()),
        },
        {
          label: MONITORS.METRICS,
          onClick: () => onGroupClick(MONITORS.METRICS.toLowerCase()),
          id: MONITORS.METRICS.toLowerCase(),
          selected: groupActions.includes(MONITORS.METRICS.toLowerCase()),
          disabled:
            groupActions.length === 1 &&
            groupActions.includes(MONITORS.METRICS.toLowerCase()),
        },
      ],
      condition: true,
    },
    {
      label: 'Sort by',
      subTitle: 'Sort by',
      items: [
        {
          label: 'Type',
          onClick: () => onSortClick(ActionsSortType.TYPE),
          id: ActionsSortType.TYPE,
          selected: currentSortId === ActionsSortType.TYPE,
        },
        {
          label: 'Action Name',
          onClick: () => onSortClick(ActionsSortType.ACTION_NAME),
          id: ActionsSortType.ACTION_NAME,
          selected: currentSortId === ActionsSortType.ACTION_NAME,
        },
        {
          label: 'Status',
          onClick: () => onSortClick(ActionsSortType.STATUS),
          id: ActionsSortType.STATUS,
          selected: currentSortId === ActionsSortType.STATUS,
        },
      ],
      condition: true,
    },
  ];

  return (
    <StyledThead>
      <StyledTh>
        <KeyvalCheckbox
          value={selectedCheckbox.length === data.length}
          onChange={() => onSelectedCheckboxChange(SELECT_ALL_CHECKBOX)}
        />
      </StyledTh>
      <StyledMainTh>
        <UnFocusAction style={{ width: 18, height: 18 }} />
        <KeyvalText size={14} weight={600} color={theme.text.white}>
          {selectedCheckbox.length > 0
            ? `${selectedCheckbox.length} selected`
            : `${data.length} ${OVERVIEW.MENU.ACTIONS}`}
        </KeyvalText>
        <ActionGroupContainer>
          <ActionsGroup actionGroups={actionGroups} />
        </ActionGroupContainer>
      </StyledMainTh>
    </StyledThead>
  );
}
