import React, { Dispatch, SetStateAction, useState } from 'react';
import styled from 'styled-components';
import { SignalUppercase } from '@/utils';
import type { DropdownOption } from '@/types';
import { Dropdown, Input, MonitoringCheckboxes } from '@/reuseable-components';

interface FilterComponentProps {
  selectedTag: DropdownOption | undefined;
  onTagSelect: (option: DropdownOption) => void;
  onSearch: (value: string) => void;
  selectedMonitors: SignalUppercase[];
  setSelectedMonitors: Dispatch<SetStateAction<SignalUppercase[]>>;
}

const Container = styled.div`
  display: flex;
  align-items: center;
  gap: 12px;
`;

const WidthConstraint = styled.div`
  width: 160px;
  margin-right: 8px;
`;

const DROPDOWN_OPTIONS = [
  { value: 'All types', id: 'all' },
  { value: 'Managed', id: 'managed' },
  { value: 'Self-hosted', id: 'self hosted' },
];

const DestinationFilterComponent: React.FC<FilterComponentProps> = ({ selectedTag, onTagSelect, onSearch, selectedMonitors, setSelectedMonitors }) => {
  const [searchTerm, setSearchTerm] = useState('');

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setSearchTerm(value);
    onSearch(value);
  };

  return (
    <Container>
      <WidthConstraint>
        <Input placeholder='Search...' icon='/icons/common/search.svg' value={searchTerm} onChange={handleSearchChange} />
      </WidthConstraint>
      <WidthConstraint>
        <Dropdown options={DROPDOWN_OPTIONS} value={selectedTag} onSelect={onTagSelect} onDeselect={() => {}} />
      </WidthConstraint>
      <MonitoringCheckboxes title='' selectedSignals={selectedMonitors} setSelectedSignals={setSelectedMonitors} />
    </Container>
  );
};

export { DestinationFilterComponent };
