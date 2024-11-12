import React from 'react';
import { Search } from './search';
<<<<<<< HEAD
import styled from 'styled-components';
import { Divider, TabList } from '@/reuseable-components';
import { AddEntity, Filters, MonitorsLegend } from '@/components';
=======
import { Filters } from './filters';
import styled from 'styled-components';
import { Divider, TabList } from '@/reuseable-components';
import { AddEntity, MonitorsLegend } from '@/components';
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866

const MenuContainer = styled.div`
  display: flex;
  align-items: center;
  margin: 20px 0;
  gap: 16px;
`;

<<<<<<< HEAD
// Aligns the "AddEntity" button to the right
=======
// Aligns the "AddEntity" button to the right.
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
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
