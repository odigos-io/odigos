import { useDrawerStore } from '@/store';
import { useNotify } from '../useNotify';
import { useMutation } from '@apollo/client';
import { ACTION, getSseTargetFromId, NOTIFICATION } from '@/utils';
import { PERSIST_SOURCE, UPDATE_K8S_ACTUAL_SOURCE } from '@/graphql';
import { useComputePlatform, useNamespace } from '../compute-platform';
import { OVERVIEW_ENTITY_TYPES, type WorkloadId, type NotificationType, PersistSourcesArray, PatchSourceRequestInput, K8sActualSource } from '@/types';

interface Params {
  onSuccess?: () => void;
  onError?: () => void;
}

export const useSourceCRUD = (params?: Params) => {
  const { setSelectedItem: setDrawerItem } = useDrawerStore((store) => store);
  const { persistNamespace } = useNamespace();
  const { startPolling } = useComputePlatform();
  const notify = useNotify();

  const notifyUser = (type: NotificationType, title: string, message: string, id?: WorkloadId) => {
    notify({
      type,
      title,
      message,
      crdType: OVERVIEW_ENTITY_TYPES.SOURCE,
      target: id ? getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.SOURCE) : undefined,
    });
  };

  const handleError = (title: string, message: string, id?: WorkloadId) => {
    notifyUser(NOTIFICATION.ERROR, title, message, id);
    params?.onError?.();
  };

  const handleComplete = (title: string, message: string, id?: WorkloadId) => {
    notifyUser(NOTIFICATION.SUCCESS, title, message, id);
    setDrawerItem(null);
    startPolling();
    params?.onSuccess?.();
  };

  const [createOrDeleteSources, cdState] = useMutation<{ persistK8sSources: boolean }>(PERSIST_SOURCE, {
    onError: (error) => handleError(ACTION.CREATE, error.message),
    onCompleted: (res, req) => {
      const namespace = req?.variables?.namespace;
      const { name, kind, selected } = req?.variables?.sources[0];

      const count = req?.variables?.sources.length;
      const action = selected ? ACTION.CREATE : ACTION.DELETE;
      const fromOrIn = selected ? 'in' : 'from';

      if (count > 1) {
        handleComplete(action, `${count} sources were ${action.toLowerCase()}d ${fromOrIn} "${namespace}"`);
      } else {
        const id = { kind, name, namespace };
        handleComplete(action, `source "${name}" was ${action.toLowerCase()}d ${fromOrIn} "${namespace}"`, id);
      }
    },
  });

  const [updateSource, uState] = useMutation<{ updateK8sActualSource: boolean }>(UPDATE_K8S_ACTUAL_SOURCE, {
    onError: (error) => handleError(ACTION.UPDATE, error.message),
    onCompleted: (res, req) => {
      const id = req?.variables?.sourceId;
      const name = id?.name;
      handleComplete(ACTION.UPDATE, `source "${name}" was updated`, id);
    },
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
    createSources: async (selectAppsList: { [key: string]: K8sActualSource[] }, futureSelectAppsList: { [key: string]: boolean }) => {
      await persistNamespaces(futureSelectAppsList);
      await persistSources(selectAppsList, true);
    },
    updateSource: async (sourceId: WorkloadId, patchSourceRequest: PatchSourceRequestInput) => await updateSource({ variables: { sourceId, patchSourceRequest } }),
    deleteSources: async (selectAppsList: { [key: string]: K8sActualSource[] }) => await persistSources(selectAppsList, false),
  };
};