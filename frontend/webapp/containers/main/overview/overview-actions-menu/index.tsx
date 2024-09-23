import React, { useState } from 'react';
import styled from 'styled-components';
import { Input, TabList } from '@/reuseable-components';

const MenuContainer = styled.div`
  width: calc(100% - 64px);
  display: flex;
  align-items: center;
  margin: 20px 0;
`;

const SearchInputContainer = styled.div`
  width: 200px;
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

export function OverviewActionMenuContainer() {
  const [searchFilter, setSearchFilter] = useState<string>('');

  return (
    <MenuContainer>
      <TabList />
      <DividerContainer>
        <Divider />
      </DividerContainer>
      <SearchInputContainer>
        <Input
          placeholder="Search "
          icon={'/icons/common/search.svg'}
          value={searchFilter}
          onChange={(e) => setSearchFilter(e.target.value)}
        />
      </SearchInputContainer>
    </MenuContainer>
  );
}
