import React, { useEffect, useState } from 'react';
import { SourcesList } from './choose-sources-list';
import { useComputePlatform, useNamespace } from '@/hooks';
import { SectionTitle, Divider } from '@/reuseable-components';
import { DropdownOption, K8sActualNamespace, K8sActualSource } from '@/types';
import { SearchAndDropdown, TogglesAndCheckboxes } from './choose-sources-menu';
import {
  SearchDropdownState,
  ToggleCheckboxState,
  SearchDropdownHandlers,
  ToggleCheckboxHandlers,
} from './choose-sources-menu/type';
import { SetupHeader } from '@/components';
import styled from 'styled-components';
import { useRouter } from 'next/navigation';
import { setSources } from '@/store';
import { useDispatch } from 'react-redux';

const ContentWrapper = styled.div`
  width: 640px;
  padding-top: 64px;
`;

const HeaderWrapper = styled.div`
  width: 100vw;
`;
export function ChooseSourcesContainer() {
  const [searchFilter, setSearchFilter] = useState('');
  const [showSelectedOnly, setShowSelectedOnly] = useState(false);
  const [selectAllCheckbox, setSelectAllCheckbox] = useState(false);
  const [futureAppsCheckbox, setFutureAppsCheckbox] = useState<{
    [key: string]: boolean;
  }>({});
  const [selectedOption, setSelectedOption] = useState<DropdownOption>();
  const [selectedItems, setSelectedItems] = useState<{
    [key: string]: K8sActualSource[];
  }>({});
  const [namespacesList, setNamespacesList] = useState<DropdownOption[]>([]);
  const [sourcesList, setSourcesList] = useState<K8sActualSource[]>([]);

  const router = useRouter();
  const dispatch = useDispatch();

  const { error, data } = useComputePlatform();
  const { data: namespacesData } = useNamespace(selectedOption?.value);

  useEffect(() => {
    data && buildNamespacesList();
  }, [data, error]);

  useEffect(() => {
    namespacesData && setSourcesList(namespacesData.k8sActualSources || []);
  }, [namespacesData]);

  useEffect(() => {
    selectAllCheckbox && selectAllSources();
  }, [selectAllCheckbox]);

  function buildNamespacesList() {
    const namespaces = data?.computePlatform?.k8sActualNamespaces || [];
    const namespacesList = namespaces.map((namespace: K8sActualNamespace) => ({
      id: namespace.name,
      value: namespace.name,
    }));

    setSelectedOption(namespacesList[0]);
    setNamespacesList(namespacesList);
  }

  function filterSources(sources: K8sActualSource[]) {
    return sources.filter((source: K8sActualSource) => {
      return (
        searchFilter === '' ||
        source.name.toLowerCase().includes(searchFilter.toLowerCase())
      );
    });
  }

  function selectAllSources() {
    if (selectedOption) {
      setSelectedItems({
        ...selectedItems,
        [selectedOption.value]: sourcesList,
      });
    }
  }

  function handleSelectItem(item: K8sActualSource) {
    if (selectedOption) {
      const currentSelectedItems = selectedItems[selectedOption.value] || [];
      if (currentSelectedItems.includes(item)) {
        const updatedSelectedItems = currentSelectedItems.filter(
          (selectedItem) => selectedItem !== item
        );
        setSelectedItems({
          ...selectedItems,
          [selectedOption.value]: updatedSelectedItems,
        });
        if (
          selectAllCheckbox &&
          updatedSelectedItems.length !== sourcesList.length
        ) {
          setSelectAllCheckbox(false);
        }
      } else {
        const updatedSelectedItems = [...currentSelectedItems, item];
        setSelectedItems({
          ...selectedItems,
          [selectedOption.value]: updatedSelectedItems,
        });
        if (updatedSelectedItems.length === sourcesList.length) {
          setSelectAllCheckbox(true);
        }
      }
    }
  }

  function getVisibleSources() {
    const allSources = sourcesList || [];
    const filteredSources = searchFilter
      ? filterSources(allSources)
      : allSources;

    return showSelectedOnly
      ? filteredSources.filter((source) =>
          selectedOption
            ? (selectedItems[selectedOption.value] || []).includes(source)
            : false
        )
      : filteredSources;
  }

  function onNextClick() {
    if (selectedOption) {
      dispatch(setSources(selectedItems[selectedOption.value] || []));
    }
    router.push('/setup/choose-destination');
  }

  const toggleCheckboxState: ToggleCheckboxState = {
    selectedAppsCount: selectedOption
      ? (selectedItems[selectedOption.value] || []).length
      : 0,
    selectAllCheckbox,
    showSelectedOnly,
    futureAppsCheckbox:
      futureAppsCheckbox[selectedOption?.value || ''] || false,
  };

  const toggleCheckboxHandlers: ToggleCheckboxHandlers = {
    setSelectAllCheckbox,
    setShowSelectedOnly,
    setFutureAppsCheckbox: (value: boolean) => {
      setFutureAppsCheckbox({
        ...futureAppsCheckbox,
        [selectedOption?.value || '']: value,
      });
    },
  };

  const searchDropdownState: SearchDropdownState = {
    selectedOption,
    searchFilter,
  };

  const searchDropdownHandlers: SearchDropdownHandlers = {
    setSelectedOption,
    setSearchFilter,
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
              disabled:
                !selectedOption ||
                (selectedItems[selectedOption.value] || []).length === 0,
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
        <Divider thickness={1} margin="16px 0" />
        <TogglesAndCheckboxes
          state={toggleCheckboxState}
          handlers={toggleCheckboxHandlers}
        />
        <Divider thickness={1} margin="16px 0 24px" />
        <SourcesList
          selectedItems={
            selectedOption ? selectedItems[selectedOption.value] || [] : []
          }
          setSelectedItems={handleSelectItem}
          items={getVisibleSources()}
        />
      </ContentWrapper>
    </>
  );
}
