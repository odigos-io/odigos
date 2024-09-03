import React from 'react';
import styled from 'styled-components';
import { SearchDropdownProps } from './type';
import { Input, Dropdown } from '@/reuseable-components';

const Container = styled.div`
  display: flex;
  gap: 8px;
  margin-top: 24px;
`;

const SearchAndDropdown: React.FC<SearchDropdownProps> = ({
  state,
  handlers,
  dropdownOptions,
}) => {
  const { selectedOption, searchFilter } = state;
  const { setSelectedOption, setSearchFilter } = handlers;

  return (
    <Container>
      <Input
        placeholder="Search for sources"
        icon={'/icons/common/search.svg'}
        value={searchFilter}
        onChange={(e) => setSearchFilter(e.target.value)}
      />
      <Dropdown
        options={dropdownOptions}
        value={selectedOption}
        onSelect={setSelectedOption}
      />
    </Container>
  );
};

export { SearchAndDropdown };
