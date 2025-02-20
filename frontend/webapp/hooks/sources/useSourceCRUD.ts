import { useEffect } from 'react';
import { useConfig } from '../config';
import { usePaginatedStore } from '@/store';
import { useNamespace } from '../compute-platform';
import { useLazyQuery, useMutation } from '@apollo/client';
import type { FetchedSource, PaginatedData, SourceUpdateInput } from '@/@types';
import { GET_SOURCE, GET_SOURCES, PERSIST_SOURCE, UPDATE_K8S_ACTUAL_SOURCE } from '@/graphql';
import { CRUD, DISPLAY_TITLES, ENTITY_TYPES, FORM_ALERTS, getSseTargetFromId, K8S_RESOURCE_KIND, NOTIFICATION_TYPE, type Source, type WorkloadId } from '@odigos/ui-utils';
import { type NamespaceSelectionFormData, type PendingItem, type SourceFormData, type SourceSelectionFormData, useNotificationStore, usePendingStore, useSetupStore } from '@odigos/ui-containers';

interface UseSourceCrud {
  sources: Source[];
  sourcesLoading: boolean;
  sourcesPaginating: boolean;
  fetchSources: (getAll?: boolean, nextPage?: string) => Promise<void>;
  fetchSourceById: (id: WorkloadId) => Promise<void>;
  persistSources: (selectAppsList: SourceSelectionFormData, futureSelectAppsList: NamespaceSelectionFormData) => Promise<void>;
  updateSource: (sourceId: WorkloadId, payload: SourceFormData) => Promise<void>;
}

const mapFetched = (items: FetchedSource[]): Source[] => {
  return items;
};

export const useSourceCRUD = (): UseSourceCrud => {
  const { data: config } = useConfig();
  const { persistNamespace } = useNamespace();
  const { addPendingItems, removePendingItems } = usePendingStore();
  const { configuredSources, setConfiguredSources } = useSetupStore();
  const { addNotification, removeNotifications } = useNotificationStore();
  const { sources, addPaginated, removePaginated, sourcesPaginating, setPaginating, setExpected } = usePaginatedStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: WorkloadId, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: ENTITY_TYPES.SOURCE, target: id ? getSseTargetFromId(id, ENTITY_TYPES.SOURCE) : undefined, hideFromHistory });
  };

  const [fetchPaginated, { loading: isFetching }] = useLazyQuery<{ computePlatform: { sources: PaginatedData<FetchedSource> } }>(GET_SOURCES, {
    fetchPolicy: 'no-cache',
  });

  const [fetchById, { loading: isFetchingById }] = useLazyQuery<{ computePlatform: { source: FetchedSource } }, { sourceId: WorkloadId }>(GET_SOURCE, {
    fetchPolicy: 'no-cache',
  });

  const fetchSources = async (getAll: boolean = true, page: string = '') => {
    setPaginating(ENTITY_TYPES.SOURCE, true);
    const { error, data } = await fetchPaginated({ variables: { nextPage: page } });

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
        setTimeout(() => fetchSources(true, nextPage), 100);
      } else if (usePaginatedStore.getState().sources.length >= usePaginatedStore.getState().sourcesExpected) {
        setPaginating(ENTITY_TYPES.SOURCE, false);
        setExpected(ENTITY_TYPES.SOURCE, 0);
      }
    }
  };

  const fetchSourceById = async (id: WorkloadId) => {
    // We have to get the boolean like this,
    // because simply using "sourcesPaginating" will contain an outdated value within this function's scope.
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

  const [persistSources, cdState] = useMutation<{ persistK8sSources: boolean }, { namespace: string; sources: Pick<Source, 'name' | 'kind' | 'selected'>[] }>(PERSIST_SOURCE, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.UPDATE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const namespace = req?.variables?.namespace;

      const hasTrueSelections = req?.variables?.sources.some(({ selected }: { selected: boolean }) => selected);
      req?.variables?.sources.forEach(({ name, kind, selected }: { name: string; kind: K8S_RESOURCE_KIND; selected: boolean }) => {
        if (!selected) {
          removeNotifications(getSseTargetFromId({ namespace, name, kind }, ENTITY_TYPES.SOURCE));
          removePaginated(ENTITY_TYPES.SOURCE, [{ namespace, name, kind }]);
          if (!hasTrueSelections) setPaginating(ENTITY_TYPES.SOURCE, false);
        }
      });

      // No fetch, we wait for SSE
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
    sources: mapFetched(sources),
    sourcesLoading: isFetching || isFetchingById || sourcesPaginating || cdState.loading || uState.loading,
    sourcesPaginating,
    fetchSources,
    fetchSourceById,

    persistSources: async (selectAppsList, futureSelectAppsList) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        const entries = Object.entries(selectAppsList);

        // this is to handle "on success" callback if there are no sources to persist,
        // and to notify use if there are source to persist
        let hasSources = false;
        let alreadyNotifiedSources = false;
        let alreadyNotifiedNamespaces = false;

        for (const [ns, items] of entries) {
          if (!!items.length) {
            hasSources = true;
            if (!alreadyNotifiedSources) {
              alreadyNotifiedSources = true;
              notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Persisting sources...', undefined, true);
            }

            // This is to stop modified events from being fetched on initial instrumentation
            setPaginating(ENTITY_TYPES.SOURCE, true);
            const exp = usePaginatedStore.getState().sourcesExpected;
            setExpected(ENTITY_TYPES.SOURCE, (!!sources.length && !exp ? sources.length : 0) + exp + items.filter((src) => src.selected).length);
          }

          const addToPendingStore: PendingItem[] = [];

          items.forEach(({ name, kind }) => {
            addToPendingStore.push({
              entityType: ENTITY_TYPES.SOURCE,
              entityId: { namespace: ns, name, kind },
            });
          });

          addPendingItems(addToPendingStore);
          await persistSources({ variables: { namespace: ns, sources: items } });
          setConfiguredSources({ ...configuredSources, [ns]: [] });
        }

        for (const [ns, items] of Object.entries(futureSelectAppsList)) {
          if (!alreadyNotifiedSources && !alreadyNotifiedNamespaces) {
            alreadyNotifiedNamespaces = true;
            notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Persisting namespaces...', undefined, true);

            // This is to stop modified events from being fetched on initial instrumentation
            setPaginating(ENTITY_TYPES.SOURCE, true);
          }

          await persistNamespace({ name: ns, futureSelected: items });
        }

        if (!hasSources) setConfiguredSources({});
      }
    },

    updateSource: async (sourceId, payload) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Updating source...', undefined, true);
        addPendingItems([{ entityType: ENTITY_TYPES.SOURCE, entityId: sourceId }]);
        await updateSourceName({ variables: { sourceId, patchSourceRequest: payload } });
      }
    },
  };
};
