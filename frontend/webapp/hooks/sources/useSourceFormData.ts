import { Dispatch, SetStateAction, useCallback, useEffect, useRef, useState } from 'react';
import { useAppStore } from '@/store';
import type { K8sActualSource } from '@/types';
import { useNamespace } from '../compute-platform';

type SelectedNamespace = string;

export type SourcesByNamespace = {
  [namespace: string]: K8sActualSource[];
};

export type FutureAppsByNamespace = {
  [namespace: string]: boolean;
};

interface UseSourceFormDataParams {
  autoSelectNamespace?: boolean;
}

export interface UseSourceFormDataResponse {
  selectedNamespace: SelectedNamespace;
  availableSources: SourcesByNamespace;
  selectedSources: SourcesByNamespace;
  selectedFutureApps: FutureAppsByNamespace;
  onSelectNamespace: (namespace: SelectedNamespace) => void;
  onSelectSource: (source: K8sActualSource, namespace?: SelectedNamespace) => void;
  onSelectFutureApps: (bool: boolean, namespace?: SelectedNamespace) => void;

  searchText: string;
  selectAll: boolean;
  selectAllForNamespace: string;
  showSelectedOnly: boolean;
  setSearchText: Dispatch<SetStateAction<string>>;
  onSelectAll: (bool: boolean, namespace?: string, isFromInterval?: boolean) => void;
  setShowSelectedOnly: Dispatch<SetStateAction<boolean>>;

  filterSources: (namespace?: string, options?: { cancelSearch?: boolean; cancelSelected?: boolean }) => K8sActualSource[];
}

export const useSourceFormData = (params?: UseSourceFormDataParams): UseSourceFormDataResponse => {
  const { autoSelectNamespace } = params || {};

  // only for "onboarding" - get unsaved values and set to state
  // (this is to persist the values when user navigates back to this page)
  const appStore = useAppStore((state) => state);

  const [selectAll, setSelectAll] = useState(false);
  const [selectAllForNamespace, setSelectAllForNamespace] = useState<SelectedNamespace>('');
  const [selectedNamespace, setSelectedNamespace] = useState<SelectedNamespace>('');
  const [availableSources, setAvailableSources] = useState<SourcesByNamespace>(appStore.availableSources);
  const [selectedSources, setSelectedSources] = useState<SourcesByNamespace>(appStore.configuredSources);
  const [selectedFutureApps, setSelectedFutureApps] = useState<FutureAppsByNamespace>(appStore.configuredFutureApps);
  const { allNamespaces, data: namespacesData } = useNamespace(selectedNamespace, false);

  useEffect(() => {
    if (!!allNamespaces?.length) {
      // auto-select the 1st namespace
      if (autoSelectNamespace) setSelectedNamespace(allNamespaces[0].name);

      // initialize all namespaces (to avoid undefined errors)
      setAvailableSources((prev) => {
        const payload = { ...prev };
        allNamespaces.forEach(({ name }) => (payload[name] = payload[name] || []));
        return payload;
      });
      setSelectedFutureApps((prev) => {
        const payload = { ...prev };
        allNamespaces.forEach(({ name }) => (payload[name] = payload[name] || false));
        return payload;
      });
    }
  }, [allNamespaces, autoSelectNamespace]);

  useEffect(() => {
    if (!!namespacesData) {
      // set available sources for current selected namespace
      const { name, k8sActualSources = [] } = namespacesData;
      setAvailableSources((prev) => ({ ...prev, [name]: k8sActualSources }));
      setSelectedSources((prev) => ({ ...prev, [name]: prev[name] || [] }));
    }
  }, [namespacesData]);

  // form filters
  const [searchText, setSearchText] = useState('');
  const [showSelectedOnly, setShowSelectedOnly] = useState(false);

  const doSelectAll = () => {
    setSelectedSources((prev) => {
      const payload = { ...prev };

      Object.entries(availableSources).forEach(([namespace, sources]) => {
        payload[namespace] = sources;
      });

      return payload;
    });
  };

  const doUnselectAll = () => {
    setSelectedSources((prev) => {
      const payload = { ...prev };

      Object.keys(availableSources).forEach((namespace) => {
        payload[namespace] = [];
      });

      return payload;
    });
  };

  const namespaceWasSelected = useRef(false);
  const onSelectAll: UseSourceFormDataResponse['onSelectAll'] = useCallback(
    (bool, namespace, isFromInterval) => {
      if (!!namespace) {
        if (!isFromInterval) namespaceWasSelected.current = selectedNamespace === namespace;
        const nsAvailableSources = availableSources[namespace];
        const nsSelectedSources = selectedSources[namespace];

        if (!nsSelectedSources && bool) {
          onSelectNamespace(namespace);
          setSelectAllForNamespace(namespace);
        } else {
          setSelectedSources((prev) => ({ ...prev, [namespace]: bool ? nsAvailableSources : [] }));
          setSelectAllForNamespace('');

          // Note: if we want to select all, but not open the expanded view, we can use the following:
          // if (!!nsAvailableSources.length && !namespaceWasSelected.current) setSelectedNamespace('');

          namespaceWasSelected.current = false;
        }
      } else {
        setSelectAll(bool);

        if (bool) {
          doSelectAll();
        } else {
          doUnselectAll();
        }
      }
    },
    [availableSources, selectedSources],
  );

  // this is to keep trying "select all" per namespace until the sources are loaded (allows for 1-click, better UX).
  // if selectedSources returns an emtpy array, it will stop to prevent inifnite loop where no availableSources ever exist for that namespace
  useEffect(() => {
    if (!!selectAllForNamespace) {
      const interval = setInterval(() => onSelectAll(true, selectAllForNamespace, true), 100);
      return () => clearInterval(interval);
    }
  }, [selectAllForNamespace, onSelectAll]);

  const onSelectNamespace: UseSourceFormDataResponse['onSelectNamespace'] = (namespace) => {
    const alreadySelected = selectedNamespace === namespace;

    setSelectedNamespace(alreadySelected ? '' : namespace);
    setAvailableSources((prev) => ({ ...prev, [namespace]: prev[namespace] || [] }));
  };

  const onSelectSource: UseSourceFormDataResponse['onSelectSource'] = (source, namespace) => {
    const id = namespace || selectedNamespace;

    if (!id) return;

    const selected = [...(selectedSources[id] || [])];
    const foundIdx = selected.findIndex(({ name, kind }) => name === source.name && kind === source.kind);

    if (foundIdx !== -1) {
      selected.splice(foundIdx, 1);
    } else {
      selected.push(source);
    }

    setSelectedSources((prev) => ({ ...prev, [id]: selected }));
    setSelectAll(false);
  };

  const onSelectFutureApps: UseSourceFormDataResponse['onSelectFutureApps'] = (bool, namespace) => {
    const id = namespace || selectedNamespace;

    if (!id) return;

    setSelectedFutureApps((prev) => ({ ...prev, [id]: bool }));
  };

  const filterSources: UseSourceFormDataResponse['filterSources'] = (namespace, options) => {
    const { cancelSearch, cancelSelected } = options || {};
    const id = namespace || selectedNamespace;

    if (!id) return [];

    const isSearchOk = (targetText: string) => cancelSearch || !searchText || targetText.toLowerCase().includes(searchText);
    const isOnlySelectedOk = (selected: Record<string, any>[], compareKey: string, target: string) => cancelSelected || !showSelectedOnly || !!selected.find((item) => item[compareKey] === target);
    const filtered = availableSources[id].filter((source) => isSearchOk(source.name) && isOnlySelectedOk(selectedSources[id], 'name', source.name));

    return filtered;
  };

  return {
    selectedNamespace,
    availableSources,
    selectedSources,
    selectedFutureApps,
    onSelectNamespace,
    onSelectSource,
    onSelectFutureApps,

    searchText,
    selectAll,
    selectAllForNamespace,
    showSelectedOnly,
    setSearchText,
    onSelectAll,
    setShowSelectedOnly,

    filterSources,
  };
};
