import React, { useState } from 'react';
import Filters from './filters';
import Monitors from './monitors';
import styled from 'styled-components';
import { AddEntityButtonDropdown } from './add-entity';
import { Input, TabList } from '@/reuseable-components';

const MenuContainer = styled.div`
  display: flex;
  align-items: center;
  margin: 20px 0;
`;

const DividerContainer = styled.div`
  height: 100%;
  display: flex;
  flex-direction: column;
`;

const Divider = styled.div`
  width: 1px;
  height: 24px;
  background-color: ${({ theme }) => theme.colors.card};
  margin: 0 16px;
`;

const SearchContainer = styled.div`
  width: 200px;
`;

const FilterContainer = styled.div`
  margin-left: 12px;
`;

const MonitorsContainer = styled.div`
  margin: 0 24px;
`;

// Aligns the AddEntityButtonDropdown to the right
const StyledAddEntityButtonDropdownWrapper = styled.div`
  margin-left: auto;
`;

export function OverviewActionMenuContainer() {
  const [search, setSearch] = useState<string>('');

  return (
    <MenuContainer>
      <TabList />

      <DividerContainer>
        <Divider />
      </DividerContainer>

      <SearchContainer>
        <Input placeholder='Search' icon='/icons/common/search.svg' value={search} onChange={(e) => setSearch(e.target.value)} />
      </SearchContainer>

      <FilterContainer>
        <Filters />
      </FilterContainer>

      <MonitorsContainer>
        <Monitors />
      </MonitorsContainer>

      <StyledAddEntityButtonDropdownWrapper>
        <AddEntityButtonDropdown />
      </StyledAddEntityButtonDropdownWrapper>
    </MenuContainer>
  );
}
