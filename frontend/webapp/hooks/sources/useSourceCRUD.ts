import { useMutation } from '@apollo/client';
import { ACTION, getSseTargetFromId } from '@/utils';
import { useAppStore, useNotificationStore } from '@/store';
import { PERSIST_SOURCE, UPDATE_K8S_ACTUAL_SOURCE } from '@/graphql';
import { useComputePlatform, useNamespace } from '../compute-platform';
import { OVERVIEW_ENTITY_TYPES, type WorkloadId, type PatchSourceRequestInput, type K8sActualSource, NOTIFICATION_TYPE } from '@/types';

interface Params {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useSourceCRUD = (params?: Params) => {
  const removeNotifications = useNotificationStore((store) => store.removeNotifications);
  const { configuredSources, setConfiguredSources } = useAppStore();

  const { persistNamespace } = useNamespace();
  const { data, refetch } = useComputePlatform();
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
    refetch();
    params?.onSuccess?.(actionType);
  };

  const [createOrDeleteSources, cdState] = useMutation<{ persistK8sSources: boolean }>(PERSIST_SOURCE, {
    onError: (error, req) => {
      const { selected } = req?.variables?.sources?.[0] || {};
      const action = selected ? ACTION.CREATE : ACTION.DELETE;

      handleError(action, error.message);
    },
    onCompleted: (res, req) => {
      const count = req?.variables?.sources.length;
      const namespace = req?.variables?.namespace;
      const { name, kind, selected } = req?.variables?.sources?.[0] || {};
      const action = selected ? ACTION.CREATE : ACTION.DELETE;

      if (count > 1) {
        handleComplete(action);
      } else {
        const id = { namespace, name, kind };
        if (!selected) removeNotifications(getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.SOURCE));
        if (!selected) setConfiguredSources({ ...configuredSources, [namespace]: configuredSources[namespace]?.filter((source) => source.name !== name) || [] });
        handleComplete(action);
      }
    },
  });

  const [updateSource, uState] = useMutation<{ updateK8sActualSource: boolean }>(UPDATE_K8S_ACTUAL_SOURCE, {
    onError: (error) => handleError(ACTION.UPDATE, error.message),
    onCompleted: () => handleComplete(ACTION.UPDATE),
  });

  const persistNamespaces = async (items: { [key: string]: boolean }) => {
    for (const [namespace, futureSelected] of Object.entries(items)) {
      await persistNamespace({ name: namespace, futureSelected });
    }
  };

  const persistSources = async (items: { [key: string]: K8sActualSource[] }, selected: boolean) => {
    for (const [namespace, sources] of Object.entries(items)) {
      await createOrDeleteSources({
        variables: {
          namespace,
          sources: sources.map((source) => ({
            kind: source.kind,
            name: source.name,
            selected,
          })),
        },
      });
    }
  };

  return {
    loading: cdState.loading || uState.loading,
    sources: data?.computePlatform.k8sActualSources || [],

    createSources: async (selectAppsList: { [key: string]: K8sActualSource[] }, futureSelectAppsList: { [key: string]: boolean }) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'creating sources...', undefined, true);
      await persistNamespaces(futureSelectAppsList);
      await persistSources(selectAppsList, true);
    },
    updateSource: async (sourceId: WorkloadId, patchSourceRequest: PatchSourceRequestInput) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'updating sources...', undefined, true);
      await updateSource({ variables: { sourceId, patchSourceRequest } });
    },
    deleteSources: async (selectAppsList: { [key: string]: K8sActualSource[] }) => {
      notifyUser(NOTIFICATION_TYPE.INFO, 'Pending', 'deleting sources...', undefined, true);
      await persistSources(selectAppsList, false);
    },
  };
};
