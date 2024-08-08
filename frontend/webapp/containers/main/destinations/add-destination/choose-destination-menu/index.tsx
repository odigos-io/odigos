import React, { useState } from 'react';
import styled from 'styled-components';
import { DropdownOption } from '@/types';
import { MonitorsTapList } from '@/components';
import { Dropdown, Input } from '@/reuseable-components';

interface FilterComponentProps {
  selectedTag: DropdownOption | undefined;
  onTagSelect: (option: DropdownOption) => void;
  onSearch: (value: string) => void;
  selectedMonitors: string[];
  onMonitorSelect: (monitor: string) => void;
}

const InputAndDropdownContainer = styled.div`
  display: flex;
  gap: 12px;
  width: 370px;
`;

const FilterContainer = styled.div`
  display: flex;
  align-items: center;
  padding: 24px 0;
`;

const DROPDOWN_OPTIONS = [
  { value: 'All types', id: 'all' },
  { value: 'Managed', id: 'managed' },
  { value: 'Self-hosted', id: 'self hosted' },
];

const DestinationFilterComponent: React.FC<FilterComponentProps> = ({
  selectedTag,
  onTagSelect,
  onSearch,
  selectedMonitors,
  onMonitorSelect,
}) => {
  const [searchTerm, setSearchTerm] = useState('');

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setSearchTerm(value);
    onSearch(value);
  };

  return (
    <FilterContainer>
      <InputAndDropdownContainer>
        <div>
          <Input
            placeholder="Search..."
            icon="/icons/common/search.svg"
            value={searchTerm}
            onChange={handleSearchChange}
          />
        </div>
        <Dropdown
          options={DROPDOWN_OPTIONS}
          selectedOption={selectedTag}
          onSelect={onTagSelect}
        />
      </InputAndDropdownContainer>
      <MonitorsTapList
        selectedMonitors={selectedMonitors}
        onMonitorSelect={onMonitorSelect}
      />
    </FilterContainer>
  );
};

export { DestinationFilterComponent };
