import { useMutation } from '@apollo/client';
import { ACTION, getSseTargetFromId } from '@/utils';
import { PERSIST_SOURCE, UPDATE_K8S_ACTUAL_SOURCE } from '@/graphql';
import { useComputePlatform, useNamespace } from '../compute-platform';
import { useAppStore, useNotificationStore, usePendingStore } from '@/store';
import { OVERVIEW_ENTITY_TYPES, type WorkloadId, type PatchSourceRequestInput, type K8sActualSource, NOTIFICATION_TYPE } from '@/types';

interface Params {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useSourceCRUD = (params?: Params) => {
  const { removeNotifications } = useNotificationStore();
  const { configuredSources, setConfiguredSources } = useAppStore();

  const { data } = useComputePlatform();
  const { persistNamespace } = useNamespace();
  const { addPendingItems } = usePendingStore();
  const { addNotification } = useNotificationStore();

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
    params?.onSuccess?.(actionType);
  };

  const [createOrDeleteSources, cdState] = useMutation<{ persistK8sSources: boolean }>(PERSIST_SOURCE, {
    onError: (error, req) => handleError('', error.message),
    onCompleted: (res, req) => {
      const count = req?.variables?.sources.length;

      if (count === 1) {
        const namespace = req?.variables?.namespace;
        const { name, kind, selected } = req?.variables?.sources?.[0] || {};
        const id = { namespace, name, kind };

        if (!selected) removeNotifications(getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.SOURCE));
        if (!selected) setConfiguredSources({ ...configuredSources, [namespace]: configuredSources[namespace]?.filter((source) => source.name !== name) || [] });
        handleComplete(selected ? ACTION.CREATE : ACTION.DELETE);
      } else {
        handleComplete('');
      }
    },
  });

  const [updateSource, uState] = useMutation<{ updateK8sActualSource: boolean }>(UPDATE_K8S_ACTUAL_SOURCE, {
    onError: (error) => handleError(ACTION.UPDATE, error.message),
    onCompleted: (res, req) => handleComplete(ACTION.UPDATE),
  });

  return {
    loading: cdState.loading || uState.loading,
    sources: data?.computePlatform.k8sActualSources || [],

    persistSources: async (selectAppsList: { [key: string]: K8sActualSource[] }, futureSelectAppsList: { [key: string]: boolean }) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Persisting sources...', undefined, true);

      for (const [namespace, sources] of Object.entries(selectAppsList)) {
        addPendingItems(sources.map(({ name, kind }) => ({ entityType: OVERVIEW_ENTITY_TYPES.SOURCE, entityId: { namespace, name, kind } })));
        await createOrDeleteSources({ variables: { namespace, sources } });
      }

      for (const [namespace, futureSelected] of Object.entries(futureSelectAppsList)) {
        await persistNamespace({ name: namespace, futureSelected });
      }
    },

    updateSource: async (sourceId: WorkloadId, patchSourceRequest: PatchSourceRequestInput) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'Updating sources...', undefined, true);
      addPendingItems([{ entityType: OVERVIEW_ENTITY_TYPES.SOURCE, entityId: sourceId }]);
      await updateSource({ variables: { sourceId, patchSourceRequest } });
    },
  };
};
