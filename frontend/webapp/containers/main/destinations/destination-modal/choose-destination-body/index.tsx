import React, { useMemo, useState } from 'react';
import styled from 'styled-components';
import { SignalUppercase } from '@/utils';
import { useDestinationTypes } from '@/hooks';
import { DestinationsList } from './destinations-list';
import { Divider, SectionTitle } from '@/reuseable-components';
import type { DropdownOption, DestinationTypeItem } from '@/types';
import { ChooseDestinationFilters } from './choose-destination-filters';

interface Props {
  onSelect: (item: DestinationTypeItem) => void;
}

const DEFAULT_MONITORS: SignalUppercase[] = ['LOGS', 'METRICS', 'TRACES'];
const DEFAULT_DROPDOWN_VALUE = { id: 'all', value: 'All types' };

const Container = styled.div`
  display: flex;
  flex-direction: column;
  gap: 24px;
`;

export const ChooseDestinationBody: React.FC<Props> = ({ onSelect }) => {
  const [searchValue, setSearchValue] = useState('');
  const [selectedMonitors, setSelectedMonitors] = useState<SignalUppercase[]>(DEFAULT_MONITORS);
  const [dropdownValue, setDropdownValue] = useState<DropdownOption>(DEFAULT_DROPDOWN_VALUE);

  const { destinations } = useDestinationTypes();

  const filteredDestinations = useMemo(() => {
    return destinations
      .map((category) => {
        const filteredItems = category.items.filter((item) => {
          const matchesSearch = searchValue ? item.displayName.toLowerCase().includes(searchValue.toLowerCase()) : true;
          const matchesDropdown = dropdownValue.id !== 'all' ? category.name === dropdownValue.id : true;
          const matchesMonitor = selectedMonitors.length ? selectedMonitors.some((monitor) => item.supportedSignals[monitor.toLowerCase()]?.supported) : true;

          return matchesSearch && matchesDropdown && matchesMonitor;
        });

        return { ...category, items: filteredItems };
      })
      .filter((category) => category.items.length > 0); // Filter out empty categories
  }, [destinations, searchValue, dropdownValue, selectedMonitors]);

  return (
    <Container>
      <SectionTitle title='Choose destination' description='Add backend destination you want to connect with Odigos.' />
      <ChooseDestinationFilters
        selectedTag={dropdownValue}
        onTagSelect={(opt) => setDropdownValue(opt)}
        onSearch={setSearchValue}
        selectedMonitors={selectedMonitors}
        setSelectedMonitors={setSelectedMonitors}
      />
      <Divider />
      <DestinationsList items={filteredDestinations} setSelectedItems={onSelect} />
    </Container>
  );
};
