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

export function ChooseSourcesContainer() {
  const [searchFilter, setSearchFilter] = useState('');
  const [showSelectedOnly, setShowSelectedOnly] = useState(false);
  const [selectAllCheckbox, setSelectAllCheckbox] = useState(false);
  const [futureAppsCheckbox, setFutureAppsCheckbox] = useState(false);
  const [selectedOption, setSelectedOption] = useState<DropdownOption>();
  const [selectedItems, setSelectedItems] = useState<K8sActualSource[]>([]);
  const [namespacesList, setNamespacesList] = useState<DropdownOption[]>([]);

  const [sourcesList, setSourcesList] = useState<K8sActualSource[]>([]);

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
    setSelectedItems(sourcesList);
  }

  function handleSelectItem(item: K8sActualSource) {
    if (selectedItems.includes(item)) {
      const updatedSelectedItems = selectedItems.filter(
        (selectedItem) => selectedItem !== item
      );
      setSelectedItems(updatedSelectedItems);
      if (
        selectAllCheckbox &&
        updatedSelectedItems.length !== sourcesList.length
      ) {
        setSelectAllCheckbox(false);
      }
    } else {
      const updatedSelectedItems = [...selectedItems, item];
      setSelectedItems(updatedSelectedItems);
      if (updatedSelectedItems.length === sourcesList.length) {
        setSelectAllCheckbox(true);
      }
    }
  }

  function getVisibleSources() {
    const allSources = sourcesList || [];
    const filteredSources = searchFilter
      ? filterSources(allSources)
      : allSources;

    return showSelectedOnly
      ? filteredSources.filter((source) => selectedItems.includes(source))
      : filteredSources;
  }

  const toggleCheckboxState: ToggleCheckboxState = {
    selectedAppsCount: selectedItems.length,
    selectAllCheckbox,
    showSelectedOnly,
    futureAppsCheckbox,
  };

  const toggleCheckboxHandlers: ToggleCheckboxHandlers = {
    setSelectAllCheckbox,
    setShowSelectedOnly,
    setFutureAppsCheckbox,
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
        selectedItems={selectedItems}
        setSelectedItems={handleSelectItem}
        items={getVisibleSources()}
      />
    </>
  );
}
