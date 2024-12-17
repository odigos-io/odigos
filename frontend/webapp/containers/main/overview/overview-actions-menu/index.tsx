import React from 'react';
import { Search } from './search';
import { Filters } from './filters';
import styled from 'styled-components';
import { AddEntity } from '@/components';
import { Divider, MonitorsIcons, TabList } from '@/reuseable-components';

const MenuContainer = styled.div`
  display: flex;
  align-items: center;
  margin: 20px 0;
  padding: 0 16px;
  gap: 16px;
`;

// Aligns the "AddEntity" button to the right.
const PushToEnd = styled.div`
  margin-left: auto;
`;

export const OverviewActionsMenu = () => {
  return (
    <MenuContainer>
      <TabList />
      <Divider orientation='vertical' length='20px' margin='0' />
      <Search />
      <Filters />
      <MonitorsIcons withLabels />

      <PushToEnd>
        <AddEntity />
      </PushToEnd>
    </MenuContainer>
  );
};
