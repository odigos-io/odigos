import { useEffect } from 'react';
import { useConfig } from '../config';
import { useNamespace } from '../namespaces';
import { useLazyQuery, useMutation } from '@apollo/client';
import { getSseTargetFromId } from '@odigos/ui-kit/functions';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import type { InstrumentationInstancesHealth, PaginatedData, SourceInstrumentInput, SourceUpdateInput } from '@/types';
import { addConditionToSources, prepareNamespacePayloads, prepareSourcePayloads } from '@/utils';
import { GET_INSTANCES, GET_SOURCE, GET_SOURCES, PERSIST_SOURCE, UPDATE_K8S_ACTUAL_SOURCE } from '@/graphql';
import { type WorkloadId, type Source, type SourceFormData, ENTITY_TYPES, STATUS_TYPE, CRUD, type Condition } from '@odigos/ui-kit/types';
import { type NamespaceSelectionFormData, type SourceSelectionFormData, useEntityStore, useInstrumentStore, useNotificationStore, usePendingStore, useSetupStore } from '@odigos/ui-kit/store';

interface UseSourceCrud {
  sources: Source[];
  sourcesLoading: boolean;
  fetchSourcesPaginated: (getAll?: boolean, nextPage?: string) => Promise<void>;
  fetchSourceById: (id: WorkloadId, bypassPaginationLoader?: boolean) => Promise<void>;
  persistSources: (selectAppsList: SourceSelectionFormData, futureSelectAppsList: NamespaceSelectionFormData) => Promise<void>;
  updateSource: (sourceId: WorkloadId, payload: SourceFormData) => Promise<void>;
}

export const useSourceCRUD = (): UseSourceCrud => {
  const { isReadonly } = useConfig();
  const { persistNamespace } = useNamespace();
  const { addNotification } = useNotificationStore();
  const { addPendingItems, removePendingItems } = usePendingStore();
  const { setInstrumentAwait, setInstrumentCount } = useInstrumentStore();
  const { setConfiguredSources, setConfiguredFutureApps } = useSetupStore();
  const { sourcesLoading, setEntitiesLoading, sources, addEntities, removeEntities } = useEntityStore();

  const notifyUser = (type: STATUS_TYPE, title: string, message: string, id?: WorkloadId, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: ENTITY_TYPES.SOURCE, target: id ? getSseTargetFromId(id, ENTITY_TYPES.SOURCE) : undefined, hideFromHistory });
  };

  const [queryByPage] = useLazyQuery<{ computePlatform: { sources: PaginatedData<Source> } }>(GET_SOURCES);
  const [queryById] = useLazyQuery<{ computePlatform: { source: Source } }, { sourceId: WorkloadId }>(GET_SOURCE);
  const [queryInstances] = useLazyQuery<{ instrumentationInstancesHealth: InstrumentationInstancesHealth[] }>(GET_INSTANCES);

  const [mutatePersistSources] = useMutation<{ persistK8sSources: boolean }, SourceInstrumentInput>(PERSIST_SOURCE, {
    onError: (error) => {
      setInstrumentCount('sourcesToCreate', 0);
      setInstrumentCount('sourcesCreated', 0);
      setInstrumentAwait(false);
      notifyUser(STATUS_TYPE.ERROR, error.name || CRUD.UPDATE, error.cause?.message || error.message);
    },
  });

  const [mutateUpdate] = useMutation<{ updateK8sActualSource: boolean }, { sourceId: WorkloadId; patchSourceRequest: SourceUpdateInput }>(UPDATE_K8S_ACTUAL_SOURCE, {
    onError: (error) => notifyUser(STATUS_TYPE.ERROR, error.name || CRUD.UPDATE, error.cause?.message || error.message),
  });

  const shouldFetchSource = (allowFetchDuringLoadTrue?: boolean) => {
    // We should not fetch if we are already fetching.
    const { sourcesLoading } = useEntityStore.getState();
    // We should not fetch while sources are being instrumented.
    const { isAwaitingInstrumentation } = useInstrumentStore.getState();

    return !isAwaitingInstrumentation && (!sourcesLoading || (sourcesLoading && allowFetchDuringLoadTrue));
  };

  const handleInstrumentationCount = (toAddCount: number, toDeleteCount: number) => {
    const { sourcesToCreate, sourcesToDelete } = useInstrumentStore.getState();

    setInstrumentCount('sourcesToDelete', sourcesToDelete + toDeleteCount);
    setInstrumentCount('sourcesToCreate', sourcesToCreate + toAddCount);
  };

  const fetchAllInstances = async () => {
    const sourcesFromStore = useEntityStore.getState().sources;
    const { data } = await queryInstances();

    if (data?.instrumentationInstancesHealth) {
      const sourcesWithInstances: Source[] = [];

      for (const instanceHealth of data.instrumentationInstancesHealth) {
        const updatedSource = addConditionToSources(instanceHealth, sourcesFromStore);
        if (updatedSource) sourcesWithInstances.push(updatedSource);
      }

      addEntities(ENTITY_TYPES.SOURCE, sourcesWithInstances);
    }
  };

  const fetchSourcesPaginated = async (getAll: boolean = true, page: string = '') => {
    if (!shouldFetchSource(!!page)) return;
    setEntitiesLoading(ENTITY_TYPES.SOURCE, true);

    const { error, data } = await queryByPage({ variables: { nextPage: page } });

    if (error) {
      notifyUser(STATUS_TYPE.ERROR, error.name || CRUD.READ, error.cause?.message || error.message);
    } else if (data?.computePlatform?.sources) {
      const { items, nextPage } = data.computePlatform.sources;

      addEntities(ENTITY_TYPES.SOURCE, items);

      if (getAll && nextPage) {
        fetchSourcesPaginated(true, nextPage);
      } else if (useEntityStore.getState().sources.length >= useInstrumentStore.getState().sourcesToCreate) {
        setEntitiesLoading(ENTITY_TYPES.SOURCE, false);
        setInstrumentCount('sourcesToCreate', 0);
        setInstrumentCount('sourcesCreated', 0);
        fetchAllInstances();
      }
    }
  };

  const fetchSourceById = async (id: WorkloadId, bypassPaginationLoader: boolean = false) => {
    if (!shouldFetchSource(bypassPaginationLoader)) return;

    const { error, data } = await queryById({ variables: { sourceId: id } });

    if (error) {
      notifyUser(STATUS_TYPE.ERROR, error.name || CRUD.READ, error.cause?.message || error.message);
    } else if (data?.computePlatform?.source) {
      addEntities(ENTITY_TYPES.SOURCE, [data.computePlatform.source]);
    }
  };

  const persistSources: UseSourceCrud['persistSources'] = async (selectAppsList, futureSelectAppsList) => {
    if (isReadonly) {
      notifyUser(STATUS_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      let alreadyNotified = false;
      const { payloads: persistSourcesPayloads, isEmpty: sourcesEmpty } = prepareSourcePayloads(selectAppsList, handleInstrumentationCount, removeEntities);
      const { payloads: persistNamespacesPayloads, isEmpty: futueAppsEmpty } = prepareNamespacePayloads(futureSelectAppsList);

      if (!sourcesEmpty && !alreadyNotified) {
        alreadyNotified = true;
        notifyUser(STATUS_TYPE.DEFAULT, 'Pending', 'Persisting sources...', undefined, true);
        setInstrumentAwait(true);
      }
      if (!futueAppsEmpty && !alreadyNotified) {
        alreadyNotified = true;
        notifyUser(STATUS_TYPE.DEFAULT, 'Pending', 'Persisting namespaces...', undefined, true);
        // TODO: estimate the number of instrumentationConfigs to create for future apps in "handleInstrumentationCount", then uncomment the below
        // setInstrumentAwait(true);
      }

      await Promise.all(persistSourcesPayloads.map((payload) => mutatePersistSources({ variables: payload })));
      setConfiguredSources({});
      await Promise.all(persistNamespacesPayloads.map(persistNamespace));
      setConfiguredFutureApps({});

      // !! no "fetch" and no "setInstrumentAwait(false)""
      // !! we should wait for SSE to handle that
    }
  };

  const updateSource: UseSourceCrud['updateSource'] = async (sourceId, payload) => {
    if (isReadonly) {
      notifyUser(STATUS_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      notifyUser(STATUS_TYPE.DEFAULT, 'Pending', 'Updating source...', undefined, true);
      addPendingItems([{ entityType: ENTITY_TYPES.SOURCE, entityId: sourceId }]);

      const patchSourceRequest: SourceUpdateInput = payload;
      const { errors } = await mutateUpdate({ variables: { sourceId, patchSourceRequest } });

      if (!errors?.length) notifyUser(STATUS_TYPE.SUCCESS, CRUD.UPDATE, `Successfully updated "${sourceId.name}" source`, sourceId);
      removePendingItems([{ entityType: ENTITY_TYPES.SOURCE, entityId: sourceId }]);

      // !! no "fetch"
      // !! we should wait for SSE to handle that
    }
  };

  useEffect(() => {
    if (!sources.length && !sourcesLoading) fetchSourcesPaginated();
  }, []);

  return {
    sources,
    sourcesLoading,
    fetchSourcesPaginated,
    fetchSourceById,
    persistSources,
    updateSource,
  };
};
