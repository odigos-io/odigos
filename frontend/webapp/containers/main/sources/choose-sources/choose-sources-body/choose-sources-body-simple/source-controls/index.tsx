import React from 'react';
import { SearchIcon } from '@/assets';
import styled from 'styled-components';
import { NamespaceDropdown } from '@/components';
import { type UseSourceFormDataResponse } from '@/hooks';
import { Badge, Checkbox, Divider, Input, SectionTitle, Text, Toggle } from '@/reuseable-components';

interface Props extends UseSourceFormDataResponse {
  isModal?: boolean;
}

const FlexContainer = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
`;

const ToggleWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 32px;
`;

export const SourceControls: React.FC<Props> = ({
  selectedNamespace,
  onSelectNamespace,
  selectedSources,
  selectedFutureApps,
  onSelectFutureApps,

  searchText,
  setSearchText,
  selectAll,
  onSelectAll,
  showSelectedOnly,
  setShowSelectedOnly,
}) => {
  const selectedAppsCount = (selectedSources[selectedNamespace || ''] || []).length;

  return (
    <>
      <SectionTitle title='Choose sources' description="Apps will be automatically instrumented, and data will be sent to the relevant APM's destinations." />

      <FlexContainer style={{ marginTop: 24 }}>
        <Input placeholder='Search for sources' icon={SearchIcon} value={searchText} onChange={(e) => setSearchText(e.target.value.toLowerCase())} />
        <NamespaceDropdown
          title=''
          value={selectedNamespace ? { value: selectedNamespace, id: selectedNamespace } : undefined}
          onSelect={({ id }) => onSelectNamespace(id)}
          onDeselect={({ id }) => onSelectNamespace(id)}
        />
      </FlexContainer>

      <Divider margin='16px 0' />

      <FlexContainer>
        <FlexContainer>
          <Text>Selected apps</Text>
          <Badge label={selectedAppsCount} filled={!!selectedAppsCount} />
        </FlexContainer>

        <ToggleWrapper>
          <Toggle title='Select all' initialValue={selectAll} onChange={onSelectAll} />
          <Toggle title='Show selected only' initialValue={showSelectedOnly} onChange={setShowSelectedOnly} />
          <Checkbox
            title='Future apps'
            tooltip='Automatically instrument all future apps'
            initialValue={!!selectedNamespace ? selectedFutureApps[selectedNamespace] : false}
            onChange={onSelectFutureApps}
          />
        </ToggleWrapper>
      </FlexContainer>

      <Divider margin='16px 0 24px' />
    </>
  );
};
