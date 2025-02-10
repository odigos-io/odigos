import { useMemo } from 'react';
import { useConfig } from '../config';
import { useMutation } from '@apollo/client';
import { useNamespace } from '../compute-platform';
import { usePaginatedStore } from '@/store';
import { PERSIST_SOURCE, UPDATE_K8S_ACTUAL_SOURCE } from '@/graphql';
import type { FetchedSource, SourceUpdateInput } from '@/types';
import { CONDITION_STATUS, CRUD, DISPLAY_TITLES, ENTITY_TYPES, FORM_ALERTS, getSseTargetFromId, K8S_RESOURCE_KIND, NOTIFICATION_TYPE, type WorkloadId } from '@odigos/ui-utils';
import { type NamespaceSelectionFormData, type PendingItem, type SourceSelectionFormData, useFilterStore, useNotificationStore, usePendingStore, useSetupStore } from '@odigos/ui-containers';

interface Params {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

interface UseSourceCrudResponse {
  loading: boolean;
  sources: FetchedSource[];
  filteredSources: FetchedSource[];

  persistSources: (selectAppsList: SourceSelectionFormData, futureSelectAppsList: NamespaceSelectionFormData) => Promise<void>;
  updateSource: (sourceId: WorkloadId, payload: SourceUpdateInput) => Promise<void>;
}

export const useSourceCRUD = (params?: Params): UseSourceCrudResponse => {
  const { persistNamespace } = useNamespace();

  const filters = useFilterStore();
  const { data: config } = useConfig();
  const { setConfiguredSources } = useSetupStore();
  const { sources, updateSource } = usePaginatedStore();
  const { addPendingItems, removePendingItems } = usePendingStore();
  const { addNotification, removeNotifications } = useNotificationStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: WorkloadId, hideFromHistory?: boolean) => {
    addNotification({
      type,
      title,
      message,
      crdType: ENTITY_TYPES.SOURCE,
      target: id ? getSseTargetFromId(id, ENTITY_TYPES.SOURCE) : undefined,
      hideFromHistory,
    });
  };

  const handleError = (actionType: string, message: string) => {
    notifyUser(NOTIFICATION_TYPE.ERROR, actionType, message);
    params?.onError?.(actionType);
  };

  const handleComplete = (actionType: string) => {
    setConfiguredSources({});
    params?.onSuccess?.(actionType);
  };

  const filtered = useMemo(() => {
    let arr = [...sources];

    if (!!filters.namespace) arr = arr.filter((source) => filters.namespace?.id === source.namespace);
    if (!!filters.types.length) arr = arr.filter((source) => !!filters.types.find((type) => type.id === source.kind));
    if (!!filters.onlyErrors) arr = arr.filter((source) => !!source.conditions?.find((cond) => cond.status === CONDITION_STATUS.FALSE));
    if (!!filters.errors.length) arr = arr.filter((source) => !!filters.errors.find((error) => !!source.conditions?.find((cond) => cond.message === error.id)));
    if (!!filters.languages.length) arr = arr.filter((source) => !!filters.languages.find((language) => !!source.containers?.find((cont) => cont.language === language.id)));

    return arr;
  }, [sources, filters]);

  const [persistSources, cdState] = useMutation<{ persistK8sSources: boolean }>(PERSIST_SOURCE, {
    onError: (error) => handleError('', error.message),
    onCompleted: (res, req) => {
      const namespace = req?.variables?.namespace;
      const count = req?.variables?.sources.length;

      req?.variables?.sources.forEach(({ name, kind, selected }: { name: string; kind: K8S_RESOURCE_KIND; selected: boolean }) => {
        if (!selected) removeNotifications(getSseTargetFromId({ namespace, name, kind }, ENTITY_TYPES.SOURCE));
      });

      if (count === 1) {
        const { selected } = req?.variables?.sources?.[0] || {};
        handleComplete(selected ? CRUD.CREATE : CRUD.DELETE);
      } else {
        handleComplete('');
      }
    },
  });

  const [updateSourceName, uState] = useMutation<{ updateK8sActualSource: boolean }>(UPDATE_K8S_ACTUAL_SOURCE, {
    onError: (error) => handleError(CRUD.UPDATE, error.message),
    onCompleted: (res, req) => {
      handleComplete(CRUD.UPDATE);

      // This is instead of toasting a k8s modified-event watcher...
      // If we do toast with a watcher, we can't guarantee an SSE will be sent for this update alone. It will definitely include SSE for all updates, even those unexpected.
      // Not that there's anything about a watcher that would break the UI, it's just that we would receive unexpected events with ridiculous amounts.
      setTimeout(() => {
        const { sourceId, patchSourceRequest } = req?.variables || {};

        updateSource(sourceId, patchSourceRequest);
        notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.UPDATE, `Successfully updated "${sourceId.name}" source`, sourceId);
        removePendingItems([{ entityType: ENTITY_TYPES.SOURCE, entityId: sourceId }]);
      }, 2000);
    },
  });

  return {
    loading: cdState.loading || uState.loading,
    sources,
    filteredSources: filtered,

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

        for (const [namespace, sources] of entries) {
          if (!!sources.length) {
            hasSources = true;
            if (!alreadyNotifiedSources) {
              alreadyNotifiedSources = true;
              notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Persisting sources...', undefined, true);
            }
          }

          const addToPendingStore: PendingItem[] = [];

          sources.forEach(({ name, kind }) => {
            addToPendingStore.push({
              entityType: ENTITY_TYPES.SOURCE,
              entityId: { namespace, name, kind },
            });
          });

          addPendingItems(addToPendingStore);
          await persistSources({ variables: { namespace, sources } });
        }

        for (const [namespace, futureSelected] of Object.entries(futureSelectAppsList)) {
          if (!alreadyNotifiedSources && !alreadyNotifiedNamespaces) {
            alreadyNotifiedNamespaces = true;
            notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Persisting namespaces...', undefined, true);
          }

          await persistNamespace({ name: namespace, futureSelected });
        }

        if (!hasSources) handleComplete('');
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
