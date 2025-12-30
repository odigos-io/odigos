import { useEffect } from 'react';
import { useConfig } from '../config';
import { useNamespace } from '../namespaces';
import { useLazyQuery, useMutation } from '@apollo/client';
import { getSseTargetFromId } from '@odigos/ui-kit/functions';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import type { SourceConditions, SourceInstrumentInput } from '@/types';
import { addAgentInjectionStatusToSources, addConditionToSources, prepareNamespacePayloads, prepareSourcePayloads } from '@/utils';
import { GET_SOURCE, GET_SOURCE_CONDITIONS, GET_SOURCE_LIBRARIES, GET_SOURCES, GET_WORKLOADS, PERSIST_SOURCES, UPDATE_K8S_ACTUAL_SOURCE } from '@/graphql';
import { type WorkloadId, type Source, type SourceFormData, EntityTypes, StatusType, Crud, InstrumentationInstanceComponent, Workload } from '@odigos/ui-kit/types';
import { type NamespaceSelectionFormData, type SourceSelectionFormData, useDataStreamStore, useEntityStore, useInstrumentStore, useNotificationStore, useSetupStore } from '@odigos/ui-kit/store';

interface UseSourceCrud {
  sources: Source[];
  sourcesLoading: boolean;
  fetchSources: () => Promise<void>;
  fetchSourceById: (id: WorkloadId, bypassPaginationLoader?: boolean) => Promise<Source | undefined>;
  fetchSourceLibraries: (id: WorkloadId) => Promise<{ data?: { instrumentationInstanceComponents: InstrumentationInstanceComponent[] } }>;
  persistSources: (selectAppsList: SourceSelectionFormData, futureSelectAppsList: NamespaceSelectionFormData) => Promise<void>;
  updateSource: (sourceId: WorkloadId, payload: SourceFormData) => Promise<void>;
}

export const useSourceCRUD = (): UseSourceCrud => {
  const { isReadonly } = useConfig();
  const { persistNamespaces } = useNamespace();
  const { addNotification } = useNotificationStore();
  const { selectedStreamName } = useDataStreamStore();
  const { setInstrumentAwait, setInstrumentCount } = useInstrumentStore();
  const { sourcesLoading, setEntitiesLoading, sources, setEntities, addEntities, removeEntities } = useEntityStore();
  const { setFetchedAllNamespaces, setAvailableSources, setConfiguredSources, setConfiguredFutureApps } = useSetupStore();

  const notifyUser = (type: StatusType, title: string, message: string, id?: WorkloadId, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: EntityTypes.Source, target: id ? getSseTargetFromId(id, EntityTypes.Source) : undefined, hideFromHistory });
  };

  const [queryAll] = useLazyQuery<{ computePlatform: { sources: Source[] } }>(GET_SOURCES);
  const [queryById] = useLazyQuery<{ computePlatform: { source: Source } }, { sourceId: WorkloadId }>(GET_SOURCE);
  const [queryOtherConditions] = useLazyQuery<{ sourceConditions: SourceConditions[] }>(GET_SOURCE_CONDITIONS);
  const [querySourceLibraries] = useLazyQuery<{ instrumentationInstanceComponents: InstrumentationInstanceComponent[] }, WorkloadId>(GET_SOURCE_LIBRARIES, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message),
  });
  const [queryWorkloads] = useLazyQuery<{ workloads: Workload[] }, { filter?: WorkloadId }>(GET_WORKLOADS);

  const [mutatePersistSources] = useMutation<{ persistK8sSources: boolean }, SourceInstrumentInput>(PERSIST_SOURCES, {
    onError: (error) => {
      setInstrumentAwait(false);
      setInstrumentCount('sourcesToCreate', 0);
      setInstrumentCount('sourcesCreated', 0);
      setInstrumentCount('sourcesToDelete', 0);
      setInstrumentCount('sourcesDeleted', 0);
      notifyUser(StatusType.Error, error.name || Crud.Update, error.cause?.message || error.message);
    },
  });

  const [mutateUpdate] = useMutation<{ updateK8sActualSource: boolean }, { sourceId: WorkloadId; patchSourceRequest: SourceFormData }>(UPDATE_K8S_ACTUAL_SOURCE, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Update, error.cause?.message || error.message),
  });

  const handleInstrumentationCount = (toAddCount: number, toDeleteCount: number) => {
    const { sourcesToCreate, sourcesToDelete } = useInstrumentStore.getState();

    // TODO: estimate the number of instrumentationConfigs to create for future-apps

    if (toDeleteCount > 0) setInstrumentCount('sourcesToDelete', sourcesToDelete + toDeleteCount);
    if (toAddCount > 0) setInstrumentCount('sourcesToCreate', sourcesToCreate + toAddCount);

    if (toDeleteCount > 0 || toAddCount > 0) setInstrumentAwait(true);
  };

  const fetchAllConditions = async (allSources: Source[]) => {
    const { data } = await queryOtherConditions();

    if (data?.sourceConditions) {
      const tempSources: Source[] = [];

      for (const item of data.sourceConditions) {
        const updatedSource = addConditionToSources(item, allSources);
        if (updatedSource) tempSources.push(updatedSource);
      }

      addEntities(EntityTypes.Source, tempSources);
    }
  };

  const fetchAllAgentInjectionStatuses = async (allSources: Source[]) => {
    const reqPayload = allSources.length === 1 ? { variables: { filter: { namespace: allSources[0].namespace, kind: allSources[0].kind, name: allSources[0].name } } } : undefined;

    const { data } = await queryWorkloads(reqPayload);
    const { workloads } = data || {};

    if (workloads) {
      const tempSources: Source[] = [];

      for (const item of workloads) {
        const updatedSource = addAgentInjectionStatusToSources(item, allSources);
        if (updatedSource) tempSources.push(updatedSource);
      }

      addEntities(EntityTypes.Source, tempSources);
    }
  };

  const fetchSources: UseSourceCrud['fetchSources'] = async () => {
    setEntitiesLoading(EntityTypes.Source, true);

    const { error, data } = await queryAll();
    const { sources: fetchedSources } = data?.computePlatform || {};

    if (error) {
      notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message);
    } else if (fetchedSources) {
      setEntities(EntityTypes.Source, fetchedSources);
      setEntitiesLoading(EntityTypes.Source, false);
      if (fetchedSources.length) {
        fetchAllAgentInjectionStatuses(fetchedSources);
        fetchAllConditions(fetchedSources);
      }
    }
  };

  const fetchSourceById: UseSourceCrud['fetchSourceById'] = async (id, bypassPaginationLoader = false): Promise<Source | undefined> => {
    const { error, data } = await queryById({ variables: { sourceId: id } });

    if (error) {
      notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message);
    } else if (data?.computePlatform?.source) {
      const { source } = data.computePlatform;
      addEntities(EntityTypes.Source, [source]);
      fetchAllAgentInjectionStatuses([source]);
      return source;
    }
  };

  const persistSources: UseSourceCrud['persistSources'] = async (selectAppsList, futureSelectAppsList) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      let alreadyNotified = false;
      const { payload: persistSourcesPayloads, isEmpty: sourcesEmpty } = prepareSourcePayloads(selectAppsList, sources, selectedStreamName, handleInstrumentationCount, removeEntities, addEntities);
      const { payload: persistNamespacesPayloads, isEmpty: futueAppsEmpty } = prepareNamespacePayloads(futureSelectAppsList, selectedStreamName);

      if (!sourcesEmpty && !alreadyNotified) {
        alreadyNotified = true;
        notifyUser(StatusType.Default, 'Pending', 'Persisting sources...', undefined, true);
      }
      if (!futueAppsEmpty && !alreadyNotified) {
        alreadyNotified = true;
        notifyUser(StatusType.Default, 'Pending', 'Persisting namespaces...', undefined, true);
      }

      await mutatePersistSources({ variables: persistSourcesPayloads });
      await persistNamespaces(persistNamespacesPayloads);

      setFetchedAllNamespaces(false);
      setAvailableSources({});
      setConfiguredSources({});
      setConfiguredFutureApps({});

      // !! no "fetch" and no "setInstrumentAwait(false)"
      // !! we should wait for SSE to handle that
    }
  };

  const updateSource: UseSourceCrud['updateSource'] = async (sourceId, payload) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      notifyUser(StatusType.Default, 'Pending', 'Updating source...', undefined, true);

      const { data } = await mutateUpdate({ variables: { sourceId, patchSourceRequest: { ...payload, currentStreamName: selectedStreamName } } });
      if (data?.updateK8sActualSource) notifyUser(StatusType.Success, Crud.Update, `Successfully updated "${sourceId.name}" source`, sourceId);

      // !! no "fetch"
      // !! we should wait for SSE to handle that
    }
  };

  useEffect(() => {
    if (!sources.length && !useEntityStore.getState().sourcesLoading) fetchSources();
  }, []);

  return {
    sources,
    sourcesLoading,
    fetchSources,
    fetchSourceById,
    fetchSourceLibraries: (payload: WorkloadId) => querySourceLibraries({ variables: payload }),
    persistSources,
    updateSource,
  };
};
