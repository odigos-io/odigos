import React, { useState } from 'react';
import styled from 'styled-components';
import { Input, TabList } from '@/reuseable-components';
import { AddEntityButtonDropdown } from '../add-entity';
import { DropdownOption } from '@/types';

const MenuContainer = styled.div`
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

// Aligns the AddEntityButtonDropdown to the right
const StyledAddEntityButtonDropdownWrapper = styled.div`
  margin-left: auto;
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
      <StyledAddEntityButtonDropdownWrapper>
        <AddEntityButtonDropdown
          onSelect={(option: DropdownOption) => {
            console.log({ option });
          }}
        />
      </StyledAddEntityButtonDropdownWrapper>
    </MenuContainer>
  );
}
