import React, { useMemo, useState } from 'react';
import { SideMenu } from '@/components';
import { useDestinationTypes } from '@/hooks';
import { DestinationsList } from '../destinations-list';
import { Body, Container, SideMenuWrapper } from '../styled';
import { Divider, SectionTitle } from '@/reuseable-components';
import { DestinationFilterComponent } from '../choose-destination-menu';
import { StepProps, DropdownOption, DestinationTypeItem } from '@/types';

interface ChooseDestinationModalBodyProps {
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

const DEFAULT_MONITORS = ['logs', 'metrics', 'traces'];
const DEFAULT_DROPDOWN_VALUE = { id: 'all', value: 'All types' };

export function ChooseDestinationModalBody({
  onSelect,
}: ChooseDestinationModalBodyProps) {
  const [searchValue, setSearchValue] = useState('');
  const [selectedMonitors, setSelectedMonitors] =
    useState<string[]>(DEFAULT_MONITORS);
  const [dropdownValue, setDropdownValue] = useState<DropdownOption>(
    DEFAULT_DROPDOWN_VALUE
  );

  const { destinations } = useDestinationTypes();

  function handleTagSelect(option: DropdownOption) {
    setDropdownValue(option);
  }

  const filteredDestinations = useMemo(() => {
    return destinations
      .map((category) => {
        const filteredItems = category.items.filter((item) => {
          const matchesSearch = searchValue
            ? item.displayName.toLowerCase().includes(searchValue.toLowerCase())
            : true;

          const matchesDropdown =
            dropdownValue.id !== 'all'
              ? category.name === dropdownValue.id
              : true;

          const matchesMonitor = selectedMonitors.length
            ? selectedMonitors.some(
                (monitor) => item.supportedSignals[monitor]?.supported
              )
            : true;

          return matchesSearch && matchesDropdown && matchesMonitor;
        });

        return { ...category, items: filteredItems };
      })
      .filter((category) => category.items.length > 0); // Filter out empty categories
  }, [destinations, searchValue, dropdownValue, selectedMonitors]);

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
        <SideMenu data={SIDE_MENU_DATA} currentStep={1} />
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
        <DestinationsList
          items={filteredDestinations}
          setSelectedItems={onSelect}
        />
      </Body>
    </Container>
  );
}
