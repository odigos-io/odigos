import { Dispatch, SetStateAction, useEffect, useMemo, useState } from 'react';
import { useAppStore } from '@/store';
import { useNamespace } from '../compute-platform';
import type { DropdownOption, K8sActualSource } from '@/types';

type SelectedNamespace = DropdownOption | undefined;

type SourcesByNamespace = {
  [namespace: string]: K8sActualSource[];
};

type FutureAppsByNamespace = {
  [namespace: string]: boolean;
};

export interface UseConnectSourcesMenuStateResponse {
  selectedNamespace: SelectedNamespace;
  onSelectNamespace: (item: DropdownOption) => void;
  selectedSources: SourcesByNamespace;
  onSelectSource: (item: K8sActualSource) => void;
  selectedFutureApps: FutureAppsByNamespace;
  onSelectFutureApps: (bool: boolean) => void;

  searchText: string;
  setSearchText: Dispatch<SetStateAction<string>>;
  selectAll: boolean;
  setSelectAll: Dispatch<SetStateAction<boolean>>;
  showSelectedOnly: boolean;
  setShowSelectedOnly: Dispatch<SetStateAction<boolean>>;

  filteredSources: K8sActualSource[];
}

export const useConnectSourcesMenuState = (): UseConnectSourcesMenuStateResponse => {
  // namespace controls/options
  const [selectedNamespace, setSelectedNamespace] = useState<SelectedNamespace>(undefined);
  const [availableSources, setAvailableSources] = useState<SourcesByNamespace>({});
  const { allNamespaces, data: namespacesData } = useNamespace(selectedNamespace?.id, false);

  // auto-select the 1st namespace
  useEffect(() => {
    if (!!allNamespaces?.length && !selectedNamespace) {
      const { name } = allNamespaces[0];
      setSelectedNamespace({ id: name, value: name });
      setAvailableSources((prev) => ({ ...prev, [name]: prev[name] || [] }));
    }
  }, [allNamespaces, selectedNamespace]);

  // set available sources for current selected namespace
  useEffect(() => {
    if (!!namespacesData) {
      const { name, k8sActualSources = [] } = namespacesData;
      setAvailableSources((prev) => ({ ...prev, [name]: k8sActualSources }));
    }
  }, [namespacesData]);

  // only for "onboarding" - get unsaved values and set to state
  // (this is to persist the values when user navigates back to this page)
  const appStore = useAppStore((state) => state);

  // form values
  const [selectedSources, setSelectedSources] = useState<SourcesByNamespace>(appStore.sources);
  const [selectedFutureApps, setSelectedFutureApps] = useState<FutureAppsByNamespace>(appStore.namespaceFutureSelectAppsList);

  // form filters
  const [searchText, setSearchText] = useState('');
  const [selectAll, setSelectAll] = useState(false);
  const [showSelectedOnly, setShowSelectedOnly] = useState(false);

  useEffect(() => {
    if (!!selectedNamespace && !!availableSources[selectedNamespace.id].length) {
      if (selectAll) {
        setSelectedSources((prev) => ({ ...prev, [selectedNamespace.id]: availableSources[selectedNamespace.id] }));
      } else {
        setSelectedSources((prev) => ({ ...prev, [selectedNamespace.id]: [] }));
      }
    }
  }, [selectAll, selectedNamespace, availableSources]);

  const onSelectNamespace = (item: DropdownOption) => {
    setSelectedNamespace(item);
    setAvailableSources((prev) => ({ ...prev, [item.id]: prev[item.id] || [] }));
  };

  const onSelectSource = (item: K8sActualSource) => {
    if (!selectedNamespace) return;

    const preAvailableSources = availableSources[selectedNamespace.id];
    const preSelectedSources = [...(selectedSources[selectedNamespace.id] || [])];
    const foundIndex = preSelectedSources.findIndex(({ name, kind }) => name === item.name && kind === item.kind);

    if (foundIndex === -1) {
      preSelectedSources.push(item);
    } else {
      preSelectedSources.splice(foundIndex, 1);
    }

    setSelectedSources((prev) => ({ ...prev, [selectedNamespace.id]: preSelectedSources }));
    setSelectAll(preSelectedSources.length === preAvailableSources.length);
  };

  const onSelectFutureApps = (bool: boolean) => {
    if (!selectedNamespace) return;

    setSelectedFutureApps((prev) => ({ ...prev, [selectedNamespace.id]: bool }));
  };

  const filteredSources = useMemo(() => {
    if (!selectedNamespace) return [];

    const preAvailableSources = availableSources[selectedNamespace.id];
    const filtered =
      !!searchText || showSelectedOnly
        ? preAvailableSources.filter(
            (source) =>
              (!searchText || source.name.toLowerCase().includes(searchText.toLowerCase())) &&
              (!showSelectedOnly || !!selectedSources[selectedNamespace.id]?.find((selected) => selected.name === source.name)),
          )
        : preAvailableSources;

    return filtered;
  }, [selectedNamespace, availableSources, searchText, showSelectedOnly, selectedSources]);

  return {
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
  };
};
