import React, { useState } from 'react';
import styled from 'styled-components';
import { DropdownOption } from '@/types';
import { MONITORS_OPTIONS } from '@/utils';
import { Checkbox, Dropdown, Input } from '@/reuseable-components';

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

const MonitorButtonsContainer = styled.div`
  display: flex;
  gap: 32px;
  margin-left: 32px;
`;

const DROPDOWN_OPTIONS = [
  { value: 'All types', id: 'all' },
  { value: 'Managed', id: 'managed' },
  { value: 'Self-hosted', id: 'self hosted' },
];

<<<<<<< HEAD
const DestinationFilterComponent: React.FC<FilterComponentProps> = ({
  selectedTag,
  selectedMonitors,
  onTagSelect,
  onSearch,
  onMonitorSelect,
}) => {
=======
const DestinationFilterComponent: React.FC<FilterComponentProps> = ({ selectedTag, selectedMonitors, onTagSelect, onSearch, onMonitorSelect }) => {
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
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
<<<<<<< HEAD
          <Input
            placeholder="Search..."
            icon="/icons/common/search.svg"
            value={searchTerm}
            onChange={handleSearchChange}
          />
        </div>
        <Dropdown
          options={DROPDOWN_OPTIONS}
          value={selectedTag}
          onSelect={onTagSelect}
          showSearch={false}
        />
      </InputAndDropdownContainer>
=======
          <Input placeholder='Search...' icon='/icons/common/search.svg' value={searchTerm} onChange={handleSearchChange} />
        </div>
        <Dropdown options={DROPDOWN_OPTIONS} value={selectedTag} onSelect={onTagSelect} onDeselect={function (option: DropdownOption): void {}} />
      </InputAndDropdownContainer>

>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
      <MonitorButtonsContainer>
        {MONITORS_OPTIONS.map((monitor) => (
          <Checkbox
            key={monitor.id}
            title={monitor.value}
            initialValue
            onChange={() => onMonitorSelect(monitor.id)}
<<<<<<< HEAD
            disabled={
              selectedMonitors.length === 1 &&
              selectedMonitors.includes(monitor.id)
            }
=======
            disabled={selectedMonitors.length === 1 && selectedMonitors.includes(monitor.id)}
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
          />
        ))}
      </MonitorButtonsContainer>
    </FilterContainer>
  );
};

export { DestinationFilterComponent };
