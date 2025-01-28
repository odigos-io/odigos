import React from 'react';
import { Search } from './search';
import { Filters } from './filters';
import styled, { useTheme } from 'styled-components';
import { AddEntity } from '@/components';
import { Divider, MonitorsIcons, TabList } from '@/reuseable-components';

const MenuContainer = styled.div`
  display: flex;
  align-items: center;
  margin: 20px 0;
  padding: 0 16px;
  gap: 8px;
`;

// Aligns the "AddEntity" button to the right.
const PushToEnd = styled.div`
  margin-left: auto;
`;

export const OverviewActionsMenu = () => {
  const theme = useTheme();

  return (
    <MenuContainer>
      <TabList />
      <Divider orientation='vertical' length='20px' margin='0' />
      <Search />
      <Filters />
      <MonitorsIcons withLabels color={theme.text.dark_grey} />

      <PushToEnd>
        <AddEntity />
      </PushToEnd>
    </MenuContainer>
  );
};
