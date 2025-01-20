import { useMemo } from 'react';
import { useMutation } from '@apollo/client';
import { PERSIST_SOURCE, UPDATE_K8S_ACTUAL_SOURCE } from '@/graphql';
import { ACTION, BACKEND_BOOLEAN, getSseTargetFromId } from '@/utils';
import { type PendingItem, useAppStore, useFilterStore, useNotificationStore, usePaginatedStore, usePendingStore } from '@/store';
import { OVERVIEW_ENTITY_TYPES, type WorkloadId, type PatchSourceRequestInput, NOTIFICATION_TYPE, type K8sActualSource, K8sResourceKind, type PersistSourcesInput } from '@/types';

interface Params {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useSourceCRUD = (params?: Params) => {
  const filters = useFilterStore();
  const { setConfiguredSources } = useAppStore();
  const { addPendingItems, removePendingItems } = usePendingStore();
  const { addNotification, removeNotifications } = useNotificationStore();
  const { sources, updateSource: updateSourceInStore } = usePaginatedStore();

  const handleError = (title: string, message: string) => {
    addNotification({ type: NOTIFICATION_TYPE.ERROR, title, message, crdType: OVERVIEW_ENTITY_TYPES.SOURCE });
    params?.onError?.(title);
  };

  const handleComplete = (actionType: string) => {
    setConfiguredSources({});
    params?.onSuccess?.(actionType);
  };

  const filtered = useMemo(() => {
    let arr = [...sources];

    if (!!filters.namespace) arr = arr.filter((source) => filters.namespace?.id === source.namespace);
    if (!!filters.types.length) arr = arr.filter((source) => !!filters.types.find((type) => type.id === source.kind));
    if (!!filters.onlyErrors) arr = arr.filter((source) => !!source.conditions?.find((cond) => cond.status === BACKEND_BOOLEAN.FALSE));
    if (!!filters.errors.length) arr = arr.filter((source) => !!filters.errors.find((error) => !!source.conditions?.find((cond) => cond.message === error.id)));
    if (!!filters.languages.length) arr = arr.filter((source) => !!filters.languages.find((language) => !!source.containers?.find((cont) => cont.language === language.id)));

    return arr;
  }, [sources, filters]);

  const [persistSources, cdState] = useMutation<{ persistK8sSources: boolean }, PersistSourcesInput>(PERSIST_SOURCE, {
    onError: (error) => handleError(error.name || ACTION.UPDATE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      const namespace = req?.variables?.namespace;
      const count = req?.variables?.sources.length;

      req?.variables?.sources.forEach(({ name, kind, selected }: { name: string; kind: K8sResourceKind; selected: boolean }) => {
        if (!selected) removeNotifications(getSseTargetFromId({ namespace, name, kind }, OVERVIEW_ENTITY_TYPES.SOURCE));
      });

      if (count === 1) {
        const { selected } = req?.variables?.sources?.[0] || {};
        handleComplete(selected ? ACTION.CREATE : ACTION.DELETE);
      } else {
        handleComplete('');
      }
    },
  });

  const [updateSourceName, uState] = useMutation<{ updateK8sActualSource: boolean }, PatchSourceRequestInput>(UPDATE_K8S_ACTUAL_SOURCE, {
    onError: (error) => handleError(error.name || ACTION.UPDATE, error.cause?.message || error.message),
    onCompleted: (res, req) => {
      handleComplete(ACTION.UPDATE);

      // This is instead of using a k8s modified-event watcher...
      // If we do use a watcher, we can't guarantee an SSE will be sent for this update alone.
      // It will definitely include SSE for all updates, that can be instrument/uninstrument, conditions changed etc.
      // Not that there's anything about a watcher that would break the UI, it's just that we would receive unexpected events with ridiculous amounts,
      // (example: instrument 5 apps, update the name of 2, then uninstrument the other 3, we would get an SSE with minimum 10 updated sources, when we expect it to show only 2 due to name change).
      setTimeout(() => {
        const sourceId = req?.variables?.sourceId;
        const patchSourceRequest = req?.variables?.patchSourceRequest;

        updateSourceInStore(sourceId, patchSourceRequest);
        removePendingItems([{ entityType: OVERVIEW_ENTITY_TYPES.SOURCE, entityId: sourceId }]);

        // Notify here, because there is no SSE for UPDATE on sources...
        addNotification({
          type: NOTIFICATION_TYPE.SUCCESS,
          title: ACTION.UPDATE,
          message: `Successfully updated "${sourceId}" source`,
          crdType: OVERVIEW_ENTITY_TYPES.SOURCE,
          target: getSseTargetFromId(sourceId, OVERVIEW_ENTITY_TYPES.SOURCE),
        });
      }, 2000);
    },
  });

  return {
    loading: cdState.loading || uState.loading,
    sources,
    filteredSources: filtered,

    persistSources: async (payload: { [key: string]: K8sActualSource[] }) => {
      // this is to handle "on success" callback if there are no sources to persist
      let hasSources = false;

      for (const [namespace, sources] of Object.entries(payload)) {
        const addToPendingStore: PendingItem[] = [];
        const sendToGql: Pick<K8sActualSource, 'name' | 'kind' | 'selected'>[] = [];

        sources.forEach(({ name, kind, selected }) => {
          addToPendingStore.push({ entityType: OVERVIEW_ENTITY_TYPES.SOURCE, entityId: { namespace, name, kind } });
          sendToGql.push({ name, kind, selected });
        });

        if (!!sendToGql.length) hasSources = true;

        addPendingItems(addToPendingStore);
        await persistSources({ variables: { namespace, sources: sendToGql } });
      }

      // this is to handle "on success" callback if there are no sources to persist
      if (!hasSources) handleComplete('');
    },

    updateSource: async (id: WorkloadId, payload: PatchSourceRequestInput['patchSourceRequest']) => {
      addPendingItems([{ entityType: OVERVIEW_ENTITY_TYPES.SOURCE, entityId: id }]);
      await updateSourceName({ variables: { sourceId: id, patchSourceRequest: payload } });
    },
  };
};
