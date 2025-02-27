import { useEffect } from 'react';
import { useConfig } from '../config';
import { usePaginatedStore } from '@/store';
import { useNamespace } from '../compute-platform';
import { useLazyQuery, useMutation } from '@apollo/client';
import { GET_SOURCE, GET_SOURCES, PERSIST_SOURCE, UPDATE_K8S_ACTUAL_SOURCE } from '@/graphql';
import type { FetchedSource, NamespaceInstrumentInput, PaginatedData, SourceInstrumentInput, SourceUpdateInput } from '@/@types';
import { CRUD, DISPLAY_TITLES, ENTITY_TYPES, FORM_ALERTS, getSseTargetFromId, NOTIFICATION_TYPE, type Source, type WorkloadId } from '@odigos/ui-utils';
import { type NamespaceSelectionFormData, type SourceFormData, type SourceSelectionFormData, useInstrumentStore, useNotificationStore, usePendingStore, useSetupStore } from '@odigos/ui-containers';

interface UseSourceCrud {
  sources: Source[];
  sourcesLoading: boolean;
  sourcesPaginating: boolean;
  fetchSources: (getAll?: boolean, nextPage?: string) => Promise<void>;
  fetchSourceById: (id: WorkloadId) => Promise<void>;
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
  const { sourcesPaginating, setPaginating, sources, addPaginated, removePaginated } = usePaginatedStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: WorkloadId, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: ENTITY_TYPES.SOURCE, target: id ? getSseTargetFromId(id, ENTITY_TYPES.SOURCE) : undefined, hideFromHistory });
  };

  const [fetchPaginated, { loading: isFetching }] = useLazyQuery<{ computePlatform: { sources: PaginatedData<FetchedSource> } }>(GET_SOURCES, {
    fetchPolicy: 'cache-and-network',
  });

  const [fetchById, { loading: isFetchingById }] = useLazyQuery<{ computePlatform: { source: FetchedSource } }, { sourceId: WorkloadId }>(GET_SOURCE, {
    fetchPolicy: 'cache-and-network',
  });

  const fetchSources = async (getAll: boolean = true, page: string = '') => {
    // We should not fetch while sources are being instrumented.
    if (useInstrumentStore.getState().isAwaitingInstrumentation) return;

    setPaginating(ENTITY_TYPES.SOURCE, true);

    const startTime = Date.now();
    const { error, data } = await fetchPaginated({ variables: { nextPage: page } });
    const endTime = Date.now();

    if (!!error) {
      addNotification({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name || CRUD.READ,
        message: error.cause?.message || error.message,
      });
    } else if (!!data?.computePlatform?.sources) {
      const { items, nextPage } = data.computePlatform.sources;

      addPaginated(ENTITY_TYPES.SOURCE, items);

      if (getAll && !!nextPage) {
        const halfSecond = 500;
        const timeElapsed = endTime - startTime;

        if (timeElapsed > halfSecond) {
          fetchSources(true, nextPage);
        } else {
          // timeout helps avoid some lag on quick paginations
          setTimeout(() => fetchSources(true, nextPage), halfSecond);
        }
      } else if (usePaginatedStore.getState().sources.length >= useInstrumentStore.getState().sourcesToCreate) {
        setPaginating(ENTITY_TYPES.SOURCE, false);
        setInstrumentCount('sourcesToCreate', 0);
        setInstrumentCount('sourcesCreated', 0);
      }
    }
  };

  const fetchSourceById = async (id: WorkloadId) => {
    // We should not fetch while sources are being instrumented.
    if (useInstrumentStore.getState().isAwaitingInstrumentation) return;
    // We should not re-fetch if we are already paginating.
    // The backend will simply restart it's "page" due to an invalid hash, which will then force a full re-fetch including this item by ID.
    if (usePaginatedStore.getState().sourcesPaginating) return;

    const { error, data } = await fetchById({ variables: { sourceId: id } });

    if (!!error) {
      addNotification({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name || CRUD.READ,
        message: error.cause?.message || error.message,
      });
    } else if (!!data?.computePlatform.source) {
      addPaginated(ENTITY_TYPES.SOURCE, [data.computePlatform.source]);
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
    if (!sources.length && !sourcesPaginating) fetchSources();
  }, []);

  return {
    sources,
    sourcesLoading: isFetching || isFetchingById || sourcesPaginating || cdState.loading || uState.loading,
    sourcesPaginating,
    fetchSources,
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
            removePaginated(
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
