import { QUERIES } from '@/utils/constants';
import {
  SelectedSources,
  ManagedSource,
  SourceSortOptions,
  Namespace,
} from '@/types';
import { useMutation, useQuery } from 'react-query';
import {
  deleteSource,
  getNamespaces,
  getSources,
  setNamespaces,
} from '@/services';
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

  const { mutate: deleteSourceMutation } = useMutation(
    ({
      namespace,
      kind,
      name,
    }: {
      namespace: string;
      kind: string;
      name: string;
    }) => deleteSource(namespace, kind, name)
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
    const data = sources?.sort((a, b) => a.name.localeCompare(b.name));
    setSortedSources(data || []);
  }, [sources]);

  async function upsertSources({ sectionData, onSuccess, onError }) {
    // Create a set of unique identifiers (name + namespace) for the sources
    const sourceIdentifiersSet = new Set(
      sources?.map(
        (source: ManagedSource) =>
          `${source.name}:${source.namespace}:${source.kind}`
      )
    );

    const updatedSectionData: SelectedSources = {};

    for (const key in sectionData) {
      const { objects, ...rest } = sectionData[key];
      const updatedObjects = objects.map((item) => {
        // Create a unique identifier for the current item
        const itemIdentifier = `${item.name}:${key}:${item.kind}`;
        return {
          ...item,
          selected: item?.selected || sourceIdentifiersSet.has(itemIdentifier),
        };
      });

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

  async function deleteSourcesHandler(sources: ManagedSource[]) {
    const promises = sources.map((source) =>
      deleteSourceMutation({
        namespace: source.namespace,
        kind: source.kind,
        name: source.name,
      })
    );
    try {
      await Promise.all(promises);
      setTimeout(() => {
        refetchSources();
      }, 1000);
    } catch (error) {
      console.log(error);
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
          const aLanguage =
            a.instrumented_application_details?.languages?.[0]?.language || '';
          const bLanguage =
            b.instrumented_application_details?.languages?.[0]?.language || '';
          return aLanguage.localeCompare(bLanguage);
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

  function filterSourcesByLanguage(languages: string[]) {
    const filtered = sources?.filter((source) =>
      languages.includes(
        source.instrumented_application_details?.languages?.[0]?.language || ''
      )
    );
    setSortedSources(filtered);
  }

  function filterSourcesByKind(kind: string[]) {
    const filtered = sources?.filter((source) =>
      kind.includes(source.kind.toLowerCase())
    );

    setSortedSources(filtered);
  }

  function filterByConditionMessage(message: string[]) {
    const sourcesWithCondition = sources?.filter((deployment) =>
      deployment.instrumented_application_details.conditions.some(
        (condition) => condition.status === 'False'
      )
    );

    const filteredSources = sourcesWithCondition?.filter((deployment) =>
      deployment.instrumented_application_details.conditions.some((condition) =>
        message.includes(condition.message)
      )
    );

    setSortedSources(filteredSources || []);
  }

  const filterByConditionStatus = (status: 'All' | 'True' | 'False') => {
    if (status === 'All') {
      setSortedSources(sources);
      return;
    }

    const filteredSources = sources?.filter((deployment) =>
      deployment.instrumented_application_details.conditions.some(
        (condition) => condition.status === status
      )
    );

    setSortedSources(filteredSources || []);
  };

  const groupErrorMessages = (): string[] => {
    const errorMessagesSet: Set<string> = new Set();

    sources?.forEach((deployment) => {
      deployment.instrumented_application_details?.conditions
        .filter((condition) => condition.status === 'False')
        .forEach((condition) => {
          errorMessagesSet.add(condition.message); // Using Set to avoid duplicates
        });
    });

    return Array.from(errorMessagesSet);
  };

  return {
    upsertSources,
    refetchSources,
    sources: sortedSources || [],
    isLoading,
    sortSources,
    filterSourcesByNamespace,
    filterSourcesByLanguage,
    filterSourcesByKind,
    instrumentedNamespaces,
    namespaces: namespaces?.namespaces || [],
    deleteSourcesHandler,
    filterByConditionStatus,
    groupErrorMessages,
    filterByConditionMessage,
  };
}
