import React from 'react';
import styled from 'styled-components';
import { K8sActualSource } from '@/types';
import { useConnectSourcesList } from '@/hooks';
import { SourcesList } from '../choose-sources-list';
import { SectionTitle, Divider } from '@/reuseable-components';
import {
  SearchAndDropdown,
  TogglesAndCheckboxes,
} from '../choose-sources-menu';
import {
  SearchDropdownState,
  ToggleCheckboxState,
  SearchDropdownHandlers,
  ToggleCheckboxHandlers,
} from '../choose-sources-menu/type';

const ContentWrapper = styled.div`
  width: 660px;
  padding-top: 64px;
`;

interface ChooseSourcesContentProps {
  stateMenu: any;
  stateHandlers: any;
  sourcesList: K8sActualSource[];
  setSourcesList: React.Dispatch<React.SetStateAction<K8sActualSource[]>>;
}

const ChooseSourcesBody: React.FC<ChooseSourcesContentProps> = ({
  stateMenu,
  stateHandlers,
  sourcesList,
  setSourcesList,
}) => {
  const { namespacesList } = useConnectSourcesList({
    stateMenu,
    setSourcesList,
  });

  function getVisibleSources() {
    const allSources = sourcesList || [];
    const filteredSources = stateMenu.searchFilter
      ? stateHandlers.filterSources(allSources)
      : allSources;

    return stateMenu.showSelectedOnly
      ? filteredSources.filter((source) =>
          stateMenu.selectedOption
            ? (
                stateMenu.selectedItems[stateMenu.selectedOption.value] || []
              ).find((selectedItem) => selectedItem.name === source.name)
            : false
        )
      : filteredSources;
  }

  const toggleCheckboxState: ToggleCheckboxState = {
    selectedAppsCount: stateMenu.selectedOption
      ? (stateMenu.selectedItems[stateMenu.selectedOption.value] || []).length
      : 0,
    selectAllCheckbox: stateMenu.selectAllCheckbox,
    showSelectedOnly: stateMenu.showSelectedOnly,
    futureAppsCheckbox:
      stateMenu.futureAppsCheckbox[stateMenu.selectedOption?.value || ''] ||
      false,
  };

  const toggleCheckboxHandlers: ToggleCheckboxHandlers = {
    setSelectAllCheckbox: stateMenu.setSelectAllCheckbox,
    setShowSelectedOnly: stateMenu.setShowSelectedOnly,
    setFutureAppsCheckbox: (value: boolean) => {
      stateMenu.setFutureAppsCheckbox({
        ...stateMenu.futureAppsCheckbox,
        [stateMenu.selectedOption?.value || '']: value,
      });
    },
  };

  const searchDropdownState: SearchDropdownState = {
    selectedOption: stateMenu.selectedOption,
    searchFilter: stateMenu.searchFilter,
  };

  const searchDropdownHandlers: SearchDropdownHandlers = {
    setSelectedOption: stateMenu.setSelectedOption,
    setSearchFilter: stateMenu.setSearchFilter,
  };

  return (
    <ContentWrapper>
      <SectionTitle
        title="Choose sources"
        description="Apps will be automatically instrumented, and data will be sent to the relevant APM's destinations."
      />
      <SearchAndDropdown
        state={searchDropdownState}
        handlers={searchDropdownHandlers}
        dropdownOptions={namespacesList}
      />
      <Divider margin="16px 0" />
      <TogglesAndCheckboxes
        state={toggleCheckboxState}
        handlers={toggleCheckboxHandlers}
      />
      <Divider margin="16px 0 24px" />
      <SourcesList
        selectedItems={
          stateMenu.selectedOption
            ? stateMenu.selectedItems[stateMenu.selectedOption.value] || []
            : []
        }
        setSelectedItems={stateHandlers.handleSelectItem}
        items={getVisibleSources()}
      />
    </ContentWrapper>
  );
};

export { ChooseSourcesBody };
