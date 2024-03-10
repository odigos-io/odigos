import React, { useState } from 'react';
import theme from '@/styles/palette';
import { Destination, DestinationsSortType } from '@/types';
import styled from 'styled-components';
import { MONITORS, OVERVIEW } from '@/utils';
import { UnFocusDestinations } from '@/assets/icons/side.menu';
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

interface DestinationsTableHeaderProps {
  data: Destination[];
  sortDestinations?: (condition: string) => void;
  filterDestinationsBySignal?: (signals: string[]) => void;
}

export function DestinationsTableHeader({
  data,
  sortDestinations,
  filterDestinationsBySignal,
}: DestinationsTableHeaderProps) {
  const [currentSortId, setCurrentSortId] = useState('');
  const [groupDestinations, setGroupDestinations] = useState([
    'traces',
    'logs',
    'metrics',
  ]);
  function onSortClick(id: string) {
    setCurrentSortId(id);
    sortDestinations && sortDestinations(id);
  }

  function onGroupClick(id: string) {
    let newGroup: string[] = [];
    if (groupDestinations.includes(id)) {
      setGroupDestinations(groupDestinations.filter((item) => item !== id));
      newGroup = groupDestinations.filter((item) => item !== id);
    } else {
      setGroupDestinations([...groupDestinations, id]);
      newGroup = [...groupDestinations, id];
    }

    filterDestinationsBySignal && filterDestinationsBySignal(newGroup);
  }
  const destinationsGroups = [
    {
      label: 'Metrics',
      subTitle: 'Display',
      items: [
        {
          label: MONITORS.TRACES,
          onClick: () => onGroupClick(MONITORS.TRACES.toLowerCase()),
          id: MONITORS.TRACES.toLowerCase(),
          selected: groupDestinations.includes(MONITORS.TRACES.toLowerCase()),
          disabled:
            groupDestinations.length === 1 &&
            groupDestinations.includes(MONITORS.TRACES.toLowerCase()),
        },
        {
          label: MONITORS.LOGS,
          onClick: () => onGroupClick(MONITORS.LOGS.toLowerCase()),
          id: MONITORS.LOGS.toLowerCase(),
          selected: groupDestinations.includes(MONITORS.LOGS.toLowerCase()),
          disabled:
            groupDestinations.length === 1 &&
            groupDestinations.includes(MONITORS.LOGS.toLowerCase()),
        },
        {
          label: MONITORS.METRICS,
          onClick: () => onGroupClick(MONITORS.METRICS.toLowerCase()),
          id: MONITORS.METRICS.toLowerCase(),
          selected: groupDestinations.includes(MONITORS.METRICS.toLowerCase()),
          disabled:
            groupDestinations.length === 1 &&
            groupDestinations.includes(MONITORS.METRICS.toLowerCase()),
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
          onClick: () => onSortClick(DestinationsSortType.TYPE),
          id: DestinationsSortType.TYPE,
          selected: currentSortId === DestinationsSortType.TYPE,
        },
        {
          label: 'Name',
          onClick: () => onSortClick(DestinationsSortType.NAME),
          id: DestinationsSortType.NAME,
          selected: currentSortId === DestinationsSortType.NAME,
        },
      ],
      condition: true,
    },
  ];

  return (
    <StyledThead>
      <StyledMainTh>
        <UnFocusDestinations style={{ width: 18, height: 18 }} />
        <KeyvalText size={14} weight={600} color={theme.text.white}>
          {`${data.length} ${OVERVIEW.MENU.DESTINATIONS}`}
        </KeyvalText>
        <ActionGroupContainer>
          <ActionsGroup actionGroups={destinationsGroups} />
        </ActionGroupContainer>
      </StyledMainTh>
    </StyledThead>
  );
}
