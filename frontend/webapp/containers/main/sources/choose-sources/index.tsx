import React, { useState } from 'react';
import styled from 'styled-components';
import { useDispatch } from 'react-redux';
import { K8sActualSource } from '@/types';
import { SetupHeader } from '@/components';
import { useRouter } from 'next/navigation';
import { SourcesList } from './choose-sources-list';
import { SectionTitle, Divider } from '@/reuseable-components';
import { setNamespaceFutureSelectAppsList, setSources } from '@/store';
import { useConnectSourcesList, useConnectSourcesMenuState } from '@/hooks';
import { SearchAndDropdown, TogglesAndCheckboxes } from './choose-sources-menu';
import {
  SearchDropdownState,
  ToggleCheckboxState,
  SearchDropdownHandlers,
  ToggleCheckboxHandlers,
} from './choose-sources-menu/type';

const ContentWrapper = styled.div`
  width: 660px;
  padding-top: 64px;
`;

const HeaderWrapper = styled.div`
  width: 100vw;
`;
export function ChooseSourcesContainer() {
  const [sourcesList, setSourcesList] = useState<K8sActualSource[]>([]);

  const { stateMenu, stateHandlers } = useConnectSourcesMenuState({
    sourcesList,
  });
  const { namespacesList } = useConnectSourcesList({
    stateMenu,
    setSourcesList,
  });

  const router = useRouter();
  const dispatch = useDispatch();

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

  function onNextClick() {
    const { selectedOption, selectedItems, futureAppsCheckbox } = stateMenu;
    if (selectedOption) {
      dispatch(setSources(selectedItems));
      dispatch(setNamespaceFutureSelectAppsList(futureAppsCheckbox));
    }
    router.push('/setup/choose-destination');
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
    <>
      <HeaderWrapper>
        <SetupHeader
          navigationButtons={[
            {
              label: 'NEXT',
              iconSrc: '/icons/common/arrow-black.svg',
              onClick: () => onNextClick(),
              variant: 'primary',
            },
          ]}
        />
      </HeaderWrapper>
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
    </>
  );
}
