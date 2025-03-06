import { useEffect } from 'react';
import { useConfig } from '../config';
import { useNamespace } from '../compute-platform';
import { useLazyQuery, useMutation } from '@apollo/client';
import { GET_INSTANCES, GET_SOURCE, GET_SOURCES, PERSIST_SOURCE, UPDATE_K8S_ACTUAL_SOURCE } from '@/graphql';
import type { FetchedSource, NamespaceInstrumentInput, PaginatedData, SourceInstrumentInput, SourceUpdateInput } from '@/@types';
import { Condition, CRUD, DISPLAY_TITLES, ENTITY_TYPES, FORM_ALERTS, getSseTargetFromId, NOTIFICATION_TYPE, sleep, type Source, type WorkloadId } from '@odigos/ui-utils';
import {
  type NamespaceSelectionFormData,
  type SourceFormData,
  type SourceSelectionFormData,
  useEntityStore,
  useInstrumentStore,
  useNotificationStore,
  usePendingStore,
  useSetupStore,
} from '@odigos/ui-containers';

interface UseSourceCrud {
  sources: Source[];
  sourcesLoading: boolean;
  fetchSourcesPaginated: (getAll?: boolean, nextPage?: string) => Promise<void>;
  fetchSourceById: (id: WorkloadId, bypassPaginationLoader?: boolean) => Promise<void>;
  persistSources: (selectAppsList: SourceSelectionFormData, futureSelectAppsList: NamespaceSelectionFormData) => Promise<void>;
  updateSource: (sourceId: WorkloadId, payload: SourceFormData) => Promise<void>;
}

export const useSourceCRUD = (): UseSourceCrud => {
  const { data: config } = useConfig();
  const { persistNamespace } = useNamespace();
  const { addNotification } = useNotificationStore();
  const { addPendingItems, removePendingItems } = usePendingStore();
  const { setInstrumentAwait, setInstrumentCount } = useInstrumentStore();
  const { setConfiguredSources, setConfiguredFutureApps } = useSetupStore();
  const { sourcesLoading, setEntitiesLoading, sources, addEntities, removeEntities } = useEntityStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: WorkloadId, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: ENTITY_TYPES.SOURCE, target: id ? getSseTargetFromId(id, ENTITY_TYPES.SOURCE) : undefined, hideFromHistory });
  };

  const [queryByPage] = useLazyQuery<{ computePlatform: { sources: PaginatedData<FetchedSource> } }>(GET_SOURCES);
  const [queryById] = useLazyQuery<{ computePlatform: { source: FetchedSource } }, { sourceId: WorkloadId }>(GET_SOURCE);
  const [queryInstances] = useLazyQuery<{ instances: { namespace: WorkloadId['namespace']; name: WorkloadId['name']; kind: WorkloadId['kind']; condition: Condition }[] }, { sourceIds: WorkloadId[] }>(
    GET_INSTANCES,
  );

  const fetchSourceById = async (id: WorkloadId, bypassPaginationLoader: boolean = false) => {
    // We should not fetch while sources are being instrumented.
    if (useInstrumentStore.getState().isAwaitingInstrumentation) return;
    // We should not re-fetch if we are already paginating.
    // The backend will simply restart it's "page" due to an invalid hash, which will then force a full re-fetch including this item by ID.
    if (useEntityStore.getState().sourcesLoading && !bypassPaginationLoader) return;

    const { error, data } = await queryById({ variables: { sourceId: id } });

    if (!!error) {
      notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.READ, error.cause?.message || error.message);
    } else if (!!data?.computePlatform.source) {
      addEntities(ENTITY_TYPES.SOURCE, [data.computePlatform.source]);
    }
  };

  const fetchAllInstances = async () => {
    const sourcesFromStore = useEntityStore.getState().sources;
    const { data } = await queryInstances({ variables: { sourceIds: sourcesFromStore.map(({ namespace, name, kind }) => ({ namespace, name, kind })) } });

    if (!!data?.instances) {
      const sourcesWithInstances: Source[] = JSON.parse(JSON.stringify(sourcesFromStore));

      for (const { namespace, name, kind, condition } of data.instances) {
        if (!!condition?.status) {
          const foundIdx = sourcesWithInstances.findIndex((x) => x.namespace === namespace && x.name === name && x.kind === kind);

          if (foundIdx !== -1) {
            if (!!sourcesWithInstances[foundIdx].conditions) {
              sourcesWithInstances[foundIdx].conditions.push(condition);
            } else {
              sourcesWithInstances[foundIdx].conditions = [condition];
            }
          }
        }
      }

      addEntities(ENTITY_TYPES.SOURCE, sourcesWithInstances);
    }
  };

  const fetchSourcesPaginated = async (getAll: boolean = true, page: string = '') => {
    // We should not fetch while sources are being instrumented.
    if (useInstrumentStore.getState().isAwaitingInstrumentation) return;
    // We should not fetch if we are already fetching.
    if (useEntityStore.getState().sourcesLoading && !page) return;

    setEntitiesLoading(ENTITY_TYPES.SOURCE, true);
    const { error, data } = await queryByPage({ variables: { nextPage: page } });

    if (!!error) {
      notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.READ, error.cause?.message || error.message);
    } else if (!!data?.computePlatform?.sources) {
      const { items, nextPage } = data.computePlatform.sources;

      addEntities(ENTITY_TYPES.SOURCE, items);

      if (getAll && !!nextPage) {
        fetchSourcesPaginated(true, nextPage);
      } else if (useEntityStore.getState().sources.length >= useInstrumentStore.getState().sourcesToCreate) {
        setEntitiesLoading(ENTITY_TYPES.SOURCE, false);
        setInstrumentCount('sourcesToCreate', 0);
        setInstrumentCount('sourcesCreated', 0);
        fetchAllInstances();
      }
    }
  };

  const [persistSources, cdState] = useMutation<{ persistK8sSources: boolean }, SourceInstrumentInput>(PERSIST_SOURCE, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.UPDATE, error.cause?.message || error.message),
    onCompleted: () => {
      // We wait for SSE
    },
  });

  const [updateSourceName, uState] = useMutation<{ updateK8sActualSource: boolean }, { sourceId: WorkloadId; patchSourceRequest: SourceUpdateInput }>(UPDATE_K8S_ACTUAL_SOURCE, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.UPDATE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      // This is instead of toasting a k8s modified-event watcher...
      // If we do toast with a watcher, we can't guarantee an SSE will be sent for this update alone. It will definitely include SSE for all updates, even those unexpected.
      // Not that there's anything about a watcher that would break the UI, it's just that we would receive unexpected events with ridiculous amounts.
      setTimeout(() => {
        const { sourceId } = req?.variables || {};

        notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.UPDATE, `Successfully updated "${sourceId.name}" source`, sourceId);
        removePendingItems([{ entityType: ENTITY_TYPES.SOURCE, entityId: sourceId }]);
      }, 1000);
    },
  });

  useEffect(() => {
    if (!sources.length && !sourcesLoading) fetchSourcesPaginated();
  }, []);

  return {
    sources,
    sourcesLoading: sourcesLoading || cdState.loading || uState.loading,
    fetchSourcesPaginated,
    fetchSourceById,

    persistSources: async (selectAppsList, futureSelectAppsList) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        let alreadyNotifiedSources = false;
        let alreadyNotifiedNamespaces = false;

        const persistSourcesPayloads: SourceInstrumentInput[] = [];
        const persistNamespacesPayloads: NamespaceInstrumentInput[] = [];

        for (const [ns, items] of Object.entries(selectAppsList)) {
          if (!!items.length) {
            if (!alreadyNotifiedSources) {
              alreadyNotifiedSources = true;
              notifyUser(NOTIFICATION_TYPE.DEFAULT, 'Pending', 'Persisting sources...', undefined, true);
              setInstrumentAwait(true);
            }

            // this is to map selected=undefined to selected=false
            const mappedItems = items.map(({ name, kind, selected }) => ({ name, kind, selected: !selected ? false : true }));

            const toDelete = mappedItems.filter((src) => !src.selected);
            const toDeleteCount = toDelete.length;
            const toAddCount = mappedItems.length - toDeleteCount;

            const { sourcesToCreate, sourcesToDelete } = useInstrumentStore.getState();
            setInstrumentCount('sourcesToDelete', sourcesToDelete + toDeleteCount);
            setInstrumentCount('sourcesToCreate', (!!toAddCount && !!sources.length && !sourcesToCreate ? sources.length : 0) + sourcesToCreate + toAddCount);

            // note: in other CRUD hooks we would use "addPendingItems" here, but for sources...
            // we instantly remove deleted items, and newly added items are not relevant for pending state.
            removeEntities(
              ENTITY_TYPES.SOURCE,
              toDelete.map(({ name, kind }) => ({ namespace: ns, name, kind })),
            );

            persistSourcesPayloads.push({ namespace: ns, sources: mappedItems });
          }
        }

        for (const [ns, futureSelected] of Object.entries(futureSelectAppsList)) {
          if (!alreadyNotifiedSources && !alreadyNotifiedNamespaces) {
            alreadyNotifiedNamespaces = true;
            notifyUser(NOTIFICATION_TYPE.DEFAULT, 'Pending', 'Persisting namespaces...', undefined, true);
            // setInstrumentAwait(true);
          }

          // TODO: estimate the number of sources to create, then uncomment "setInstrumentAwait" above

          persistNamespacesPayloads.push({ name: ns, futureSelected });
        }

        for await (const payload of persistSourcesPayloads) await persistSources({ variables: payload });
        setConfiguredSources({});

        for await (const payload of persistNamespacesPayloads) await persistNamespace(payload);
        setConfiguredFutureApps({});
      }
    },

    updateSource: async (sourceId, payload) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        notifyUser(NOTIFICATION_TYPE.DEFAULT, 'Pending', 'Updating source...', undefined, true);
        addPendingItems([{ entityType: ENTITY_TYPES.SOURCE, entityId: sourceId }]);
        await updateSourceName({ variables: { sourceId, patchSourceRequest: payload } });
      }
    },
  };
};
