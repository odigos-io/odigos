import { QUERIES } from '@/utils/constants';
import {
  SelectedSources,
  ManagedSource,
  SourceSortOptions,
  Namespace,
} from '@/types';
import { useMutation, useQuery } from 'react-query';
import { getNamespaces, getSources, setNamespaces } from '@/services';
import { useEffect, useState } from 'react';

export function useSources() {
  const [instrumentedNamespaces, setInstrumentedNamespaces] = useState<
    Namespace[]
  >([]);
  const {
    data: sources,
    isLoading,
    refetch: refetchSources,
  } = useQuery<ManagedSource[]>([QUERIES.API_SOURCES], getSources);

  const { data: namespaces } = useQuery<{ namespaces: Namespace[] }>(
    [QUERIES.API_NAMESPACES],
    getNamespaces
  );

  useEffect(() => {
    if (namespaces?.namespaces && sources) {
      const instrumented = namespaces.namespaces.map((item) => {
        const totalApps =
          sources?.filter((source) => source.namespace === item.name).length ||
          0;
        return {
          ...item,
          totalApps,
          selected: false,
        };
      });

      setInstrumentedNamespaces(instrumented);
    }
  }, [namespaces, sources]);

  const [sortedSources, setSortedSources] = useState<
    ManagedSource[] | undefined
  >(undefined);

  const { mutateAsync } = useMutation((body: SelectedSources) =>
    setNamespaces(body)
  );

  useEffect(() => {
    setSortedSources(sources || []);
  }, [sources]);

  async function upsertSources({ sectionData, onSuccess, onError }) {
    const sourceNamesSet = new Set(
      sources?.map((source: ManagedSource) => source.name)
    );
    const updatedSectionData: SelectedSources = {};

    for (const key in sectionData) {
      const { objects, ...rest } = sectionData[key];
      const updatedObjects = objects.map((item) => ({
        ...item,
        selected: item?.selected || sourceNamesSet.has(item.name),
      }));

      updatedSectionData[key] = {
        ...rest,
        objects: updatedObjects,
      };
    }

    try {
      await mutateAsync(updatedSectionData);
      if (onSuccess) {
        onSuccess();
      }
    } catch (error) {
      if (onError) {
        onError(error);
      }
    }
  }

  function sortSources(condition: string) {
    const sorted = [...(sources || [])].sort((a, b) => {
      switch (condition) {
        case SourceSortOptions.NAME:
          return a.name.localeCompare(b.name);
        case SourceSortOptions.NAMESPACE:
          return a.namespace.localeCompare(b.namespace);
        case SourceSortOptions.KIND:
          return a.kind.localeCompare(b.kind);
        case SourceSortOptions.LANGUAGE:
          return a.languages[0].language.localeCompare(b.languages[0].language);
        default:
          return 0;
      }
    });
    setSortedSources(sorted);
  }

  function filterSourcesByNamespace(namespaces: string[]) {
    const filtered = sources?.filter((source) =>
      namespaces.includes(source.namespace)
    );
    setSortedSources(filtered);
  }

  function filterSourcesByKind(kind: string[]) {
    const filtered = sources?.filter((source) =>
      kind.includes(source.kind.toLowerCase())
    );

    setSortedSources(filtered);
  }

  return {
    upsertSources,
    refetchSources,
    sources: sortedSources || [],
    isLoading,
    sortSources,
    filterSourcesByNamespace,
    filterSourcesByKind,
    instrumentedNamespaces,
    namespaces: namespaces?.namespaces || [],
  };
}
