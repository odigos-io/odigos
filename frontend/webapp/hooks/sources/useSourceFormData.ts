import { Dispatch, SetStateAction, useCallback, useEffect, useRef, useState } from 'react';
import { useAppStore } from '@/store';
import type { K8sActualSource } from '@/types';
import { useNamespace } from '../compute-platform';

type SelectedNamespace = string;
type SelectedSource = Pick<K8sActualSource, 'name' | 'kind' | 'selected' | 'numberOfInstances'>;

interface UseSourceFormDataParams {
  autoSelectNamespace?: boolean;
}

export interface UseSourceFormDataResponse {
  namespacesLoading: boolean;
  recordedInitialSources: { [namespace: SelectedNamespace]: SelectedSource[] };
  filterNamespaces: (options?: { cancelSearch?: boolean }) => [SelectedNamespace, SelectedSource[]][];
  filterSources: (namespace?: SelectedNamespace, options?: { cancelSearch?: boolean; cancelSelected?: boolean }) => SelectedSource[];
  getApiSourcesPayload: () => { [namespace: SelectedNamespace]: SelectedSource[] };
  getApiFutureAppsPayload: () => { [namespace: SelectedNamespace]: boolean };

  selectedNamespace: SelectedNamespace;
  onSelectNamespace: (namespace: SelectedNamespace) => void;
  selectedSources: { [namespace: SelectedNamespace]: SelectedSource[] };
  onSelectSource: (source: SelectedSource, namespace?: SelectedNamespace) => void;
  selectedFutureApps: { [namespace: SelectedNamespace]: boolean };
  onSelectFutureApps: (bool: boolean, namespace?: SelectedNamespace) => void;

  searchText: string;
  setSearchText: Dispatch<SetStateAction<string>>;
  showSelectedOnly: boolean;
  setShowSelectedOnly: Dispatch<SetStateAction<boolean>>;
  selectAllForNamespace: SelectedNamespace;
  onSelectAll: (bool: boolean, namespace?: SelectedNamespace, isFromInterval?: boolean) => void;
}

export const useSourceFormData = (params?: UseSourceFormDataParams): UseSourceFormDataResponse => {
  const { autoSelectNamespace } = params || {};

  // only for "onboarding" - get unsaved values and set to state
  // (this is to persist the values when user navigates back to this page)
  const appStore = useAppStore();

  const [selectAllForNamespace, setSelectAllForNamespace] = useState<SelectedNamespace>('');
  const [selectedNamespace, setSelectedNamespace] = useState<SelectedNamespace>('');
  const [selectedSources, setSelectedSources] = useState<UseSourceFormDataResponse['selectedSources']>(appStore.configuredSources);
  const [selectedFutureApps, setSelectedFutureApps] = useState<UseSourceFormDataResponse['selectedFutureApps']>(appStore.configuredFutureApps);

  const { allNamespaces, data: singleNamespace, loading: namespacesLoading } = useNamespace(selectedNamespace);
  // Keeps intial values fetched from API, so we can later filter the user-specific-selections, therebey minimizing the amount of data sent to the API on "persist sources".
  const [recordedInitialSources, setRecordedInitialSources] = useState<UseSourceFormDataResponse['selectedSources']>(appStore.availableSources);

  useEffect(() => {
    if (!!allNamespaces?.length) {
      // initialize all states (to avoid undefined errors)
      setRecordedInitialSources((prev) => {
        const payload = { ...prev };
        allNamespaces.forEach(({ name }) => (payload[name] = payload[name] || []));
        return payload;
      });
      setSelectedSources((prev) => {
        const payload = { ...prev };
        allNamespaces.forEach(({ name }) => (payload[name] = payload[name] || []));
        return payload;
      });
      setSelectedFutureApps((prev) => {
        const payload = { ...prev };
        allNamespaces.forEach(({ name, selected }) => (payload[name] = payload[name] || selected || false));
        return payload;
      });
      // auto-select the 1st namespace
      if (autoSelectNamespace) setSelectedNamespace(allNamespaces[0].name);
    }
  }, [allNamespaces, autoSelectNamespace]);

  useEffect(() => {
    if (!!singleNamespace) {
      // initialize sources for this namespace
      const { name, k8sActualSources = [] } = singleNamespace;
      setRecordedInitialSources((prev) => ({ ...prev, [name]: k8sActualSources }));
      setSelectedSources((prev) => ({ ...prev, [name]: !!prev[name].length ? prev[name] : k8sActualSources }));
    }
  }, [singleNamespace]);

  // form filters
  const [searchText, setSearchText] = useState('');
  const [showSelectedOnly, setShowSelectedOnly] = useState(false);

  const onSelectAll: UseSourceFormDataResponse['onSelectAll'] = useCallback(
    (bool, namespace, isFromInterval) => {
      if (!!namespace) {
        // When clicking "select all" on a namespace

        if (!isFromInterval && bool) {
          onSelectNamespace(namespace);
          setSelectAllForNamespace(namespace);
        } else {
          setSelectedSources((prev) => ({ ...prev, [namespace]: selectedSources[namespace].map((source) => ({ ...source, selected: bool })) }));
          setSelectAllForNamespace('');
        }
      } else {
        // When clicking "select all" on all namespaces

        setSelectedSources((prev) => {
          const payload = { ...prev };

          Object.entries(payload).forEach(([namespace, sources]) => {
            payload[namespace] = sources.map((source) => ({ ...source, selected: bool }));
          });

          return payload;
        });
      }
    },
    [selectedSources],
  );

  // This is to keep trying "select all" per namespace, until the sources are loaded (allows for 1-click, better UX).
  useEffect(() => {
    if (!!selectAllForNamespace) {
      const interval = setInterval(() => onSelectAll(true, selectAllForNamespace, true), 100);
      return () => clearInterval(interval);
    }
  }, [selectAllForNamespace, onSelectAll]);

  const onSelectNamespace: UseSourceFormDataResponse['onSelectNamespace'] = (namespace) => {
    setSelectedNamespace((prev) => (prev === namespace ? '' : namespace));
  };

  const onSelectSource: UseSourceFormDataResponse['onSelectSource'] = (source, namespace) => {
    const id = namespace || selectedNamespace;

    if (!id) return;

    const arr = [...(selectedSources[id] || [])];
    const foundIdx = arr.findIndex(({ name, kind }) => name === source.name && kind === source.kind);

    if (foundIdx !== -1) {
      // Replace the item with a new object to avoid mutating a possibly read-only object
      const updatedItem = { ...arr[foundIdx], selected: !arr[foundIdx].selected };
      arr[foundIdx] = updatedItem;
    } else {
      arr.push({ ...source, selected: true });
    }

    setSelectedSources((prev) => ({ ...prev, [id]: arr }));
  };

  const onSelectFutureApps: UseSourceFormDataResponse['onSelectFutureApps'] = (bool, namespace) => {
    const id = namespace || selectedNamespace;

    if (!id) return;

    setSelectedFutureApps((prev) => ({ ...prev, [id]: bool }));
  };

  const filterNamespaces: UseSourceFormDataResponse['filterNamespaces'] = (options) => {
    const { cancelSearch } = options || {};
    const namespaces = Object.entries(selectedSources);

    const isSearchOk = (targetText: string) => cancelSearch || !searchText || targetText.toLowerCase().includes(searchText);

    return namespaces.filter(([namespace]) => isSearchOk(namespace));
  };

  const filterSources: UseSourceFormDataResponse['filterSources'] = (namespace, options) => {
    const { cancelSearch, cancelSelected } = options || {};
    const id = namespace || selectedNamespace;

    if (!id) return [];

    const isSearchOk = (targetText: string) => cancelSearch || !searchText || targetText.toLowerCase().includes(searchText);
    const isOnlySelectedOk = (sources: Record<string, any>[], compareKey: string, target: string) =>
      cancelSelected || !showSelectedOnly || !!sources.find((item) => item[compareKey] === target && item.selected);

    return selectedSources[id].filter((source) => isSearchOk(source.name) && isOnlySelectedOk(selectedSources[id], 'name', source.name));
  };

  // This is to filter the user-specific-selections, therebey minimizing the amount of data sent to the API on "persist sources".
  const getApiSourcesPayload: UseSourceFormDataResponse['getApiSourcesPayload'] = () => {
    const payload: UseSourceFormDataResponse['selectedSources'] = {};

    Object.entries(selectedSources).forEach(([namespace, sources]) => {
      sources.forEach((source) => {
        const foundInitial = recordedInitialSources[namespace]?.find((initialSource) => initialSource.name === source.name && initialSource.kind === source.kind);

        if (foundInitial?.selected !== source.selected) {
          if (!payload[namespace]) payload[namespace] = [];
          payload[namespace].push(source);
        }
      });
    });

    return payload;
  };

  // This is to filter the user-specific-selections, therebey minimizing the amount of data sent to the API on "persist namespaces".
  const getApiFutureAppsPayload: UseSourceFormDataResponse['getApiFutureAppsPayload'] = () => {
    const payload: UseSourceFormDataResponse['selectedFutureApps'] = {};

    Object.entries(selectedFutureApps).forEach(([namespace, selected]) => {
      const foundInitial = allNamespaces?.find((ns) => ns.name === namespace);

      if (foundInitial?.selected !== selected) {
        payload[namespace] = selected;
      }
    });

    return payload;
  };

  return {
    namespacesLoading,
    recordedInitialSources,
    filterNamespaces,
    filterSources,
    getApiSourcesPayload,
    getApiFutureAppsPayload,

    selectedNamespace,
    onSelectNamespace,
    selectedSources,
    onSelectSource,
    selectedFutureApps,
    onSelectFutureApps,

    searchText,
    setSearchText,
    showSelectedOnly,
    setShowSelectedOnly,
    selectAllForNamespace,
    onSelectAll,
  };
};
