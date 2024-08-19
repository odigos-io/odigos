import React, { useState } from 'react';

import { SideMenu } from '@/components';
import { DestinationsList } from '../destinations-list';
import { Body, Container, SideMenuWrapper } from '../styled';
import { Divider, SectionTitle } from '@/reuseable-components';
import { DestinationFilterComponent } from '../choose-destination-menu';
import { DestinationTypeItem, DropdownOption, StepProps } from '@/types';

interface ChooseDestinationModalBodyProps {
  data: DestinationTypeItem[];
  onSelect: (item: DestinationTypeItem) => void;
}

const SIDE_MENU_DATA: StepProps[] = [
  {
    title: 'DESTINATIONS',
    state: 'active',
    stepNumber: 1,
  },
  {
    title: 'CONNECTION',
    state: 'disabled',
    stepNumber: 2,
  },
];

export function ChooseDestinationModalBody({
  data,
  onSelect,
}: ChooseDestinationModalBodyProps) {
  const [searchValue, setSearchValue] = useState('');
  const [selectedMonitors, setSelectedMonitors] = useState<string[]>([]);
  const [dropdownValue, setDropdownValue] = useState<DropdownOption>({
    id: 'all',
    value: 'All types',
  });

  function handleTagSelect(option: DropdownOption) {
    setDropdownValue(option);
  }

  function filterData() {
    let filteredData = data;

    if (searchValue) {
      filteredData = filteredData.filter((item) =>
        item.displayName.toLowerCase().includes(searchValue.toLowerCase())
      );
    }

    if (dropdownValue.id !== 'all') {
      filteredData = filteredData.filter(
        (item) => item.category === dropdownValue.id
      );
    }

    if (selectedMonitors.length) {
      filteredData = filteredData.filter((item) =>
        selectedMonitors.some(
          (monitor) =>
            item.supportedSignals[monitor as keyof typeof item.supportedSignals]
              .supported
        )
      );
    }

    return filteredData;
  }

  function onMonitorSelect(monitor: string) {
    if (selectedMonitors.includes(monitor)) {
      setSelectedMonitors(selectedMonitors.filter((item) => item !== monitor));
    } else {
      setSelectedMonitors([...selectedMonitors, monitor]);
    }
  }

  return (
    <Container>
      <SideMenuWrapper>
        <SideMenu data={SIDE_MENU_DATA} />
      </SideMenuWrapper>
      <Body>
        <SectionTitle
          title="Choose destination"
          description="Add backend destination you want to connect with Odigos."
        />
        <DestinationFilterComponent
          selectedTag={dropdownValue}
          onTagSelect={handleTagSelect}
          onSearch={setSearchValue}
          selectedMonitors={selectedMonitors}
          onMonitorSelect={onMonitorSelect}
        />
        <Divider margin="0 0 24px 0" />
        <DestinationsList items={filterData()} setSelectedItems={onSelect} />
      </Body>
    </Container>
  );
}
