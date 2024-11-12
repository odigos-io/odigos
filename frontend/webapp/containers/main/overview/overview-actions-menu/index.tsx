import React from 'react';
import { Search } from './search';
import { Filters } from './filters';
import styled from 'styled-components';
import { Divider, TabList } from '@/reuseable-components';
import { AddEntity, MonitorsLegend } from '@/components';

const MenuContainer = styled.div`
  display: flex;
  align-items: center;
  margin: 20px 0;
  gap: 16px;
`;

// Aligns the "AddEntity" button to the right.
const PushToEnd = styled.div`
  margin-left: auto;
`;

export function OverviewActionMenuContainer() {
  return (
    <MenuContainer>
      <TabList />
      <Divider orientation='vertical' length='20px' margin='0' />
      <Search />
      <Filters />
      <MonitorsLegend />

      <PushToEnd>
        <AddEntity />
      </PushToEnd>
    </MenuContainer>
  );
}
