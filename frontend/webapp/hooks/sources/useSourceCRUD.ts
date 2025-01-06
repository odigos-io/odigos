import { useMutation } from '@apollo/client';
import { ACTION, getSseTargetFromId } from '@/utils';
import { PERSIST_SOURCE, UPDATE_K8S_ACTUAL_SOURCE } from '@/graphql';
import { useComputePlatform, useNamespace } from '../compute-platform';
import { type PendingItem, useAppStore, useNotificationStore, usePendingStore } from '@/store';
import { OVERVIEW_ENTITY_TYPES, type WorkloadId, type PatchSourceRequestInput, NOTIFICATION_TYPE, type K8sActualSource } from '@/types';

interface Params {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useSourceCRUD = (params?: Params) => {
  const { persistNamespace } = useNamespace();
  const { data, refetch } = useComputePlatform();
  const { setConfiguredSources } = useAppStore();
  const { addPendingItems, removePendingItems } = usePendingStore();
  const { addNotification, removeNotifications } = useNotificationStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: WorkloadId, hideFromHistory?: boolean) => {
    addNotification({
      type,
      title,
      message,
      crdType: OVERVIEW_ENTITY_TYPES.SOURCE,
      target: id ? getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.SOURCE) : undefined,
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

  const [createOrDeleteSources, cdState] = useMutation<{ persistK8sSources: boolean }>(PERSIST_SOURCE, {
    onError: (error) => handleError('', error.message),
    onCompleted: (res, req) => {
      const namespace = req?.variables?.namespace;
      const count = req?.variables?.sources.length;

      req?.variables?.sources.forEach(({ name, kind, selected }) => {
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

  const [updateSource, uState] = useMutation<{ updateK8sActualSource: boolean }>(UPDATE_K8S_ACTUAL_SOURCE, {
    onError: (error) => handleError(ACTION.UPDATE, error.message),
    onCompleted: (res, req) => {
      handleComplete(ACTION.UPDATE);

      // This is instead of using a k8s modified-event watcher...
      // If we do use a watcher, we can't guarantee an SSE will be sent for this update alone.
      // It will definitely include SSE for all updates, that can be instrument/uninstrument, conditions changed etc.
      // Not that there's anything about a watcher that would break the UI, it's just that we would receive unexpected events with ridiculous amounts,
      // (example: instrument 5 apps, update the name of 2, then uninstrument the other 3, we would get an SSE with minimum 10 updated sources, when we expect it to show only 2 due to name change).
      setTimeout(() => {
        const id = req?.variables?.sourceId;

        refetch();
        notifyUser(NOTIFICATION_TYPE.SUCCESS, ACTION.UPDATE, 'Successfully updated 1 source', id);
        removePendingItems([{ entityType: OVERVIEW_ENTITY_TYPES.SOURCE, entityId: id }]);
      }, 2000);
    },
  });

  return {
    loading: cdState.loading || uState.loading,
    sources: data?.computePlatform.k8sActualSources || [],

    persistSources: async (selectAppsList: { [key: string]: K8sActualSource[] }, futureSelectAppsList: { [key: string]: boolean }) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Persisting sources...', undefined, true);

      // this is to handle "on success" callback if there are no sources to persist
      let hasSources = false;

      for (const [namespace, sources] of Object.entries(selectAppsList)) {
        const addToPendingStore: PendingItem[] = [];
        const sendToGql: Pick<K8sActualSource, 'name' | 'kind' | 'selected'>[] = [];

        sources.forEach(({ name, kind, selected }) => {
          addToPendingStore.push({ entityType: OVERVIEW_ENTITY_TYPES.SOURCE, entityId: { namespace, name, kind } });
          sendToGql.push({ name, kind, selected });
        });

        if (!!sendToGql.length) hasSources = true;

        addPendingItems(addToPendingStore);
        await createOrDeleteSources({ variables: { namespace, sources: sendToGql } });
      }

      for (const [namespace, futureSelected] of Object.entries(futureSelectAppsList)) {
        await persistNamespace({ name: namespace, futureSelected });
      }

      if (!hasSources) handleComplete('');
    },

    updateSource: async (sourceId: WorkloadId, patchSourceRequest: PatchSourceRequestInput) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Updating source...', undefined, true);
      addPendingItems([{ entityType: OVERVIEW_ENTITY_TYPES.SOURCE, entityId: sourceId }]);
      await updateSource({ variables: { sourceId, patchSourceRequest } });
    },
  };
};
