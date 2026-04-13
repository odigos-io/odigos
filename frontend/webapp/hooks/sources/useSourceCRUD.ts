import { useEffect } from 'react';
import { useConfig } from '../config';
import { useNamespace } from '../namespaces';
import { useLazyQuery, useMutation } from '@apollo/client';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { getIdFromSseTarget, getSseTargetFromId } from '@odigos/ui-kit/functions';
import type { NamespaceInstrumentInput, SourceInstrumentInput, WorkloadResponse } from '@/types';
import { mapWorkloadToSource, sortSources, prepareNamespacePayloads, prepareSourcePayloads, mapConditionsToConditionArray } from '@/utils';
import { GET_PEER_SOURCES, GET_SOURCE, GET_SOURCE_LIBRARIES, GET_WORKLOADS, GET_WORKLOADS_BY_IDS, PERSIST_SOURCES, UPDATE_K8S_ACTUAL_SOURCE } from '@/graphql';
import { type WorkloadId, type Source, type SourceFormData, type PeerSources, EntityTypes, StatusType, Crud, InstrumentationInstanceComponent, PersistSourceInput } from '@odigos/ui-kit/types';
import {
  type NamespaceSelectionFormData,
  type SourceSelectionFormData,
  useDataStreamStore,
  useEntityStore,
  useNotificationStore,
  useSetupStore,
  useProgressStore,
  ProgressKeys,
} from '@odigos/ui-kit/store';

const MAX_INDIVIDUAL_FETCH = 50;

interface UseSourceCrud {
  sources: Source[];
  sourcesLoading: boolean;
  fetchSources: () => Promise<void>;
  fetchSourcesByTargets: (targets: string[]) => Promise<void>;
  fetchSourceById: (id: WorkloadId) => Promise<Source | undefined>;
  fetchSourceLibraries: (id: WorkloadId) => Promise<{ data?: { instrumentationInstanceComponents: InstrumentationInstanceComponent[] } }>;
  fetchPeerSources: (serviceName: string) => Promise<{ data?: { peerSources: PeerSources } }>;
  persistSources: (selectAppsList: SourceSelectionFormData, futureSelectAppsList: NamespaceSelectionFormData) => Promise<void>;
  persistSourcesV2: (payload: PersistSourceInput) => Promise<{ error?: string } | undefined>;
  updateSource: (sourceId: WorkloadId, payload: SourceFormData) => Promise<void>;
}

export const useSourceCRUD = (): UseSourceCrud => {
  const { isReadonly } = useConfig();
  const { persistNamespaces } = useNamespace();
  const { addNotification } = useNotificationStore();
  const { selectedStreamName } = useDataStreamStore();
  const { setProgress, resetProgress } = useProgressStore();
  const { setConfiguredSources, setConfiguredFutureApps } = useSetupStore();
  const { sourcesLoading, setEntitiesLoading, sources, setEntities, addEntities, removeEntities } = useEntityStore();

  const notifyUser = (type: StatusType, title: string, message: string, id?: WorkloadId, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: EntityTypes.Source, target: id ? getSseTargetFromId(id, EntityTypes.Source) : undefined, hideFromHistory });
  };

  const [queryById] = useLazyQuery<{ computePlatform: { source: Source } }, { sourceId: WorkloadId }>(GET_SOURCE);
  const [querySourceLibraries] = useLazyQuery<{ instrumentationInstanceComponents: InstrumentationInstanceComponent[] }, WorkloadId>(GET_SOURCE_LIBRARIES, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message),
  });
  const [queryPeerSources] = useLazyQuery<{ peerSources: PeerSources }, { serviceName: string }>(GET_PEER_SOURCES, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message),
  });
  const [queryWorkloads] = useLazyQuery<{ workloads: WorkloadResponse[] }, { filter?: { markedForInstrumentation?: boolean } & Partial<WorkloadId> }>(GET_WORKLOADS);
  const [queryWorkloadsByIds] = useLazyQuery<{ workloadsByIds: WorkloadResponse[] }, { ids: { namespace: string; kind: string; name: string }[] }>(GET_WORKLOADS_BY_IDS);

  const [mutatePersistSources] = useMutation<{ persistK8sSources: boolean }, SourceInstrumentInput>(PERSIST_SOURCES, {
    onError: (error) => {
      resetProgress(ProgressKeys.Instrumenting);
      resetProgress(ProgressKeys.Uninstrumenting);
      notifyUser(StatusType.Error, error.name || Crud.Update, error.cause?.message || error.message);
    },
  });

  const [mutateUpdate] = useMutation<{ updateK8sActualSource: boolean }, { sourceId: WorkloadId; patchSourceRequest: SourceFormData }>(UPDATE_K8S_ACTUAL_SOURCE, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Update, error.cause?.message || error.message),
  });

  const handleInstrumentationCount = (toAddCount: number, toDeleteCount: number) => {
    const { progress } = useProgressStore.getState();

    // TODO: estimate the number of instrumentationConfigs to create for future-apps

    if (toAddCount > 0)
      setProgress(ProgressKeys.Instrumenting, {
        total: (progress[ProgressKeys.Instrumenting]?.total || 0) + toAddCount,
        current: progress[ProgressKeys.Instrumenting]?.current || 0,
      });
    if (toDeleteCount > 0)
      setProgress(ProgressKeys.Uninstrumenting, {
        total: (progress[ProgressKeys.Uninstrumenting]?.total || 0) + toDeleteCount,
        current: progress[ProgressKeys.Uninstrumenting]?.current || 0,
      });
  };

  const fetchSources: UseSourceCrud['fetchSources'] = async () => {
    setEntitiesLoading(EntityTypes.Source, true);

    const { error, data } = await queryWorkloads({ variables: { filter: { markedForInstrumentation: true } } });

    if (error) {
      notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message);
    } else if (data?.workloads) {
      const mappedSources = sortSources(data.workloads.map(mapWorkloadToSource));
      setEntities(EntityTypes.Source, mappedSources);
    }

    setEntitiesLoading(EntityTypes.Source, false);
  };

  const fetchSourcesByTargets: UseSourceCrud['fetchSourcesByTargets'] = async (targets) => {
    const ids = targets.map((t) => getIdFromSseTarget(t, EntityTypes.Source) as WorkloadId).filter((id) => id.namespace && id.name && id.kind);

    if (ids.length === 0) return;

    if (ids.length > MAX_INDIVIDUAL_FETCH) {
      await fetchSources();
      return;
    }

    const { error, data } = await queryWorkloadsByIds({
      variables: { ids: ids.map(({ namespace, kind, name }) => ({ namespace, kind, name })) },
    });

    if (error) {
      notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message);
    } else if (data?.workloadsByIds) {
      addEntities(EntityTypes.Source, data.workloadsByIds.map(mapWorkloadToSource));
    }
  };

  const fetchSourceById: UseSourceCrud['fetchSourceById'] = async (id): Promise<Source | undefined> => {
    const { error: sourceError, data: sourceData } = await queryById({ variables: { sourceId: id } });

    if (sourceError) {
      notifyUser(StatusType.Error, sourceError.name || Crud.Read, sourceError.cause?.message || sourceError.message);
      return undefined;
    }

    if (!sourceData?.computePlatform?.source) return undefined;

    const { source } = sourceData.computePlatform;

    const { data: workloadData } = await queryWorkloads({ variables: { filter: { namespace: id.namespace, kind: id.kind, name: id.name } } });
    const workload = workloadData?.workloads?.[0];

    if (workload) {
      const enrichedSource: Source = {
        ...source,
        conditions: mapConditionsToConditionArray(workload.conditions),
        workloadOdigosHealthStatus: workload.workloadOdigosHealthStatus,
        podsAgentInjectionStatus: workload.podsAgentInjectionStatus,
        rollbackOccurred: workload.rollbackOccurred,
      };
      addEntities(EntityTypes.Source, [enrichedSource]);
      return enrichedSource;
    }

    addEntities(EntityTypes.Source, [source]);
    return source;
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

      setConfiguredSources({});
      setConfiguredFutureApps({});

      // !! no "fetch" and no "setInstrumentAwait(false)"
      // !! we should wait for SSE to handle that
    }
  };

  const persistSourcesV2 = async (input: PersistSourceInput) => {
    const entries = Object.entries(input);
    const errors: string[] = [];

    for await (const [_, inputs] of entries) {
      const workloadSources: SourceInstrumentInput = { sources: [] };
      const namespaceSources: NamespaceInstrumentInput = { namespaces: [] };

      for (const source of inputs) {
        if (source.name && source.kind) {
          // workload source
          workloadSources.sources.push({
            namespace: source.namespace,
            name: source.name,
            kind: source.kind,
            selected: source.selected,
            currentStreamName: source.currentStreamName,
          });
        } else {
          // namespace source
          namespaceSources.namespaces.push({
            namespace: source.namespace,
            selected: source.selected,
            currentStreamName: source.currentStreamName,
          });
        }
      }

      if (namespaceSources.namespaces.length > 0) await persistNamespaces(namespaceSources);
      if (workloadSources.sources.length > 0) await mutatePersistSources({ variables: workloadSources });
    }

    return {
      error: errors.length > 0 ? `Failed to persist sources in ${errors.length} clusters: ${errors.join(', ')}` : undefined,
    };
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
    fetchSourcesByTargets,
    fetchSourceById,
    fetchSourceLibraries: (payload: WorkloadId) => querySourceLibraries({ variables: payload }),
    fetchPeerSources: (serviceName: string) => queryPeerSources({ variables: { serviceName } }),
    persistSources,
    persistSourcesV2,
    updateSource,
  };
};
