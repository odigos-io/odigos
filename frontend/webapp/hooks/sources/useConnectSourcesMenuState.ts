import { useEffect, useState } from 'react';
import { useAppStore } from '@/store';
import { DropdownOption, K8sActualSource } from '@/types';

export const useConnectSourcesMenuState = ({ sourcesList }) => {
  const [searchFilter, setSearchFilter] = useState('');
  const [showSelectedOnly, setShowSelectedOnly] = useState(false);
  const [selectAllCheckbox, setSelectAllCheckbox] = useState(false);
  const [selectedOption, setSelectedOption] = useState<DropdownOption>();
  const [futureAppsCheckbox, setFutureAppsCheckbox] = useState<{
    [key: string]: boolean;
  }>({});
  const [selectedItems, setSelectedItems] = useState<{
    [key: string]: K8sActualSource[];
  }>({});

  const sources = useAppStore((state) => state.sources);
  const namespaceFutureSelectAppsList = useAppStore(
    (state) => state.namespaceFutureSelectAppsList
  );

  useEffect(() => {
    sources && setSelectedItems(sources);
    namespaceFutureSelectAppsList &&
      setFutureAppsCheckbox(namespaceFutureSelectAppsList);
  }, [namespaceFutureSelectAppsList, sources]);

  useEffect(() => {
    selectAllCheckbox && selectAllSources();
  }, [selectAllCheckbox]);

  function selectAllSources() {
    if (selectedOption) {
      setSelectedItems({
        ...selectedItems,
        [selectedOption.value]: sourcesList,
      });
    }
  }

  function filterSources(sources: K8sActualSource[]) {
    return sources.filter((source: K8sActualSource) => {
      return (
        searchFilter === '' ||
        source.name.toLowerCase().includes(searchFilter.toLowerCase())
      );
    });
  }

  function handleSelectItem(item: K8sActualSource) {
    if (selectedOption) {
      const currentSelectedItems = selectedItems[selectedOption.value] || [];

      const isItemSelected = currentSelectedItems.some(
        (currentSelectedItem) =>
          currentSelectedItem.name === item.name &&
          currentSelectedItem.kind === item.kind
      );

      if (isItemSelected) {
        const updatedSelectedItems = currentSelectedItems.filter(
          (selectedItem) =>
            JSON.stringify(selectedItem) !== JSON.stringify(item)
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

  return {
    stateMenu: {
      searchFilter,
      setSearchFilter,
      showSelectedOnly,
      setShowSelectedOnly,
      selectAllCheckbox,
      setSelectAllCheckbox,
      selectedOption,
      setSelectedOption,
      futureAppsCheckbox,
      setFutureAppsCheckbox,
      selectedItems,
    },
    stateHandlers: {
      handleSelectItem,
      filterSources,
    },
  };
};
