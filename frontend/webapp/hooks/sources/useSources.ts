import { QUERIES } from '@/utils/constants';
import { SelectedSources, ManagedSource, SourceSortOptions } from '@/types';
import { useMutation, useQuery } from 'react-query';
import { getSources, setNamespaces } from '@/services';
import { useEffect, useState } from 'react';

export function useSources() {
  const { data: sources, isLoading } = useQuery<ManagedSource[]>(
    [QUERIES.API_SOURCES],
    getSources
  );

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

  return {
    upsertSources,
    sources: sortedSources || [],
    isLoading,
    sortSources,
  };
}
