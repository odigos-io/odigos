import React from 'react';
import { ModalBody } from '@/styles';
import styled from 'styled-components';
import { SourcesList } from './sources-list';
import { NamespaceDropdown } from '@/components';
import { UseConnectSourcesMenuStateResponse } from '@/hooks';
import { SectionTitle, Divider, Input, Toggle, Checkbox, Text, Badge } from '@/reuseable-components';

interface Props extends UseConnectSourcesMenuStateResponse {
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

export const ChooseSourcesBodySimple: React.FC<Props> = ({
  isModal = false,

  selectedNamespace,
  onSelectNamespace,
  selectedSources,
  onSelectSource,
  selectedFutureApps,
  onSelectFutureApps,

  searchText,
  setSearchText,
  selectAll,
  setSelectAll,
  showSelectedOnly,
  setShowSelectedOnly,

  filteredSources,
}) => {
  const selectedAppsCount = (selectedSources[selectedNamespace?.id || ''] || []).length;

  return (
    <ModalBody>
      <SectionTitle title='Choose sources' description="Apps will be automatically instrumented, and data will be sent to the relevant APM's destinations." />

      <FlexContainer style={{ marginTop: 24 }}>
        <Input placeholder='Search for sources' icon={'/icons/common/search.svg'} value={searchText} onChange={(e) => setSearchText(e.target.value)} />
        <NamespaceDropdown title='' value={selectedNamespace} onSelect={onSelectNamespace} onDeselect={onSelectNamespace} />
      </FlexContainer>

      <Divider margin='16px 0' />

      <FlexContainer>
        <FlexContainer>
          <Text>Selected apps</Text>
          <Badge label={selectedAppsCount} filled={!!selectedAppsCount} />
        </FlexContainer>

        <ToggleWrapper>
          <Toggle title='Select all' initialValue={selectAll} onChange={setSelectAll} />
          <Toggle title='Show selected only' initialValue={showSelectedOnly} onChange={setShowSelectedOnly} />
          <Checkbox
            title='Future apps'
            tooltip='Automatically instrument all future apps'
            initialValue={!!selectedNamespace ? selectedFutureApps[selectedNamespace.id] : false}
            onChange={onSelectFutureApps}
          />
        </ToggleWrapper>
      </FlexContainer>

      <Divider margin='16px 0 24px' />

      <SourcesList isModal={isModal} availableSources={filteredSources} selectedSources={selectedSources[selectedNamespace?.id || ''] || []} setSelectedSources={onSelectSource} />
    </ModalBody>
  );
};
