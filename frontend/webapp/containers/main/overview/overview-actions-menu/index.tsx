import React from 'react';
import Search from './search';
import styled from 'styled-components';
import { Divider, TabList } from '@/reuseable-components';
import { AddEntity, Filters, MonitorsLegend } from '@/components';

const MenuContainer = styled.div`
  display: flex;
  align-items: center;
  margin: 20px 0;
`;

const FilterContainer = styled.div`
  margin-left: 12px;
`;

const MonitorsContainer = styled.div`
  margin-left: 24px;
`;

// Aligns the AddEntityButtonDropdown to the right
const StyledAddEntityButtonDropdownWrapper = styled.div`
  margin-left: auto;
`;

export function OverviewActionMenuContainer() {
  return (
    <MenuContainer>
      <TabList />
      <Divider orientation='vertical' length='20px' margin='0 16px' />
      <Search />

      <FilterContainer>
        <Filters />
      </FilterContainer>

      <MonitorsContainer>
        <MonitorsLegend />
      </MonitorsContainer>

      <StyledAddEntityButtonDropdownWrapper>
        <AddEntity />
      </StyledAddEntityButtonDropdownWrapper>
    </MenuContainer>
  );
}
