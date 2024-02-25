'use client';
import React, { useEffect, useMemo } from 'react';
import { SETUP } from '@/utils/constants';
import { KeyvalDropDown, KeyvalSearchInput, KeyvalText } from '@/design.system';
import { TapList } from '../tap.list/tap.list';
import {
  DropdownWrapper,
  SourcesOptionMenuWrapper,
  TapsWrapper,
} from './destination.option.menu.styled';

type DestinationOptionMenuProps = {
  setDropdownData: (data: any) => void;
  data: any;
  searchFilter: string;
  setSearchFilter: (filter: string) => void;
  monitoringOption: any;
  setMonitoringOption: (option: any) => void;
};

type DropdownItem = {
  id: number;
  label: string;
};

type MonitoringOption = {
  id: string;
  tapped: boolean;
};

export function DestinationOptionMenu({
  setDropdownData,
  data = [],
  searchFilter,
  setSearchFilter,
  monitoringOption,
  setMonitoringOption,
}: DestinationOptionMenuProps) {
  const dropdownData = useMemo(() => {
    const options = [
      { id: 'all', label: SETUP.ALL },
      ...data?.map(({ name }) => ({ id: name, label: name })),
    ];

    return options;
  }, [data]);

  useEffect(() => {
    setDropdownData(dropdownData[0]);
  }, [dropdownData]);

  function handleDropDownChange(item: DropdownItem) {
    setDropdownData({ id: item?.id, label: item.label });
  }

  function handleTapClick(id: MonitoringOption) {
    const tappedMonitors = monitoringOption.filter(
      (monitor: any) => monitor.tapped
    );

    const currentMonitorIndex = monitoringOption.findIndex(
      (monitor) => monitor.id === id
    );
    // if only one monitor is tapped and the tapped monitor is clicked, do nothing
    const isOnlyOneMonitorTapped = tappedMonitors.length === 1;
    const isTappedMonitor = monitoringOption[currentMonitorIndex].tapped;
    if (isOnlyOneMonitorTapped && isTappedMonitor) return;

    const newMonitoringOption = monitoringOption.map((monitor) => {
      if (monitor.id === id) {
        return { ...monitor, tapped: !monitor.tapped };
      }
      return monitor;
    });

    setMonitoringOption(newMonitoringOption);
  }

  return (
    <SourcesOptionMenuWrapper>
      <KeyvalSearchInput
        value={searchFilter}
        onChange={(e) => setSearchFilter(e.target.value)}
      />

      <DropdownWrapper>
        <KeyvalText size={14}>{SETUP.MENU.TYPE}</KeyvalText>
        <KeyvalDropDown
          width={180}
          data={dropdownData}
          value={dropdownData[0]}
          onChange={handleDropDownChange}
        />
      </DropdownWrapper>
      <TapsWrapper>
        <KeyvalText size={14}>{SETUP.MENU.MONITORING}</KeyvalText>
        <TapList list={monitoringOption} onClick={handleTapClick} />
      </TapsWrapper>
    </SourcesOptionMenuWrapper>
  );
}
