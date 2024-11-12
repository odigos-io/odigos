import React from 'react';
import { ModalBody } from '@/styles';
import styled from 'styled-components';
import { NamespaceDropdown } from '@/components';
import { SourcesList } from '../../choose-sources-list';
import { SectionTitle, Divider, Input, Toggle, Checkbox, Text, Badge } from '@/reuseable-components';
import { UseConnectSourcesMenuStateResponse } from '@/hooks';

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

const SourcesListWrapper = styled.div<{ isModal: boolean }>`
  display: flex;
  align-items: center;
  flex-direction: column;
  gap: 12px;
  max-height: ${({ isModal }) => (isModal ? 'calc(100vh - 548px)' : 'calc(100vh - 360px)')};
  height: fit-content;
  padding-bottom: ${({ isModal }) => (isModal ? '48px' : '0')};
  overflow-y: scroll;
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

      <SourcesListWrapper isModal={isModal}>
        <SourcesList items={filteredSources} selectedItems={selectedNamespace ? selectedSources[selectedNamespace.id] || [] : []} setSelectedItems={onSelectSource} />
      </SourcesListWrapper>
    </ModalBody>
  );
};
