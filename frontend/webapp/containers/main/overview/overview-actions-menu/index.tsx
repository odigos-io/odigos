import React from 'react';
import { Search } from './search';
import { Filters } from './filters';
import { AddEntity } from '@/components';
import { TabList } from '@/reuseable-components';
import styled, { useTheme } from 'styled-components';
import { Divider, MonitorsIcons } from '@odigos/ui-components';

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
