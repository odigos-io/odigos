import { useConfig } from '../config';
import { useMutation } from '@apollo/client';
import { RECOVER_FROM_ROLLBACK, RESTART_POD, RESTART_WORKLOADS } from '@/graphql';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { getSseTargetFromId } from '@odigos/ui-kit/functions';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { type WorkloadId, EntityTypes, StatusType, Crud } from '@odigos/ui-kit/types';

interface UseWorkloadUtils {
  restartWorkloads: (sourceIds: WorkloadId[]) => Promise<void>;
  restartPod: (namespace: string, name: string) => Promise<void>;
  recoverFromRollback: (sourceId: WorkloadId) => Promise<void>;
}

export const useWorkloadUtils = (): UseWorkloadUtils => {
  const { isReadonly } = useConfig();
  const { addNotification } = useNotificationStore();

  const notifyUser = (type: StatusType, title: string, message: string, id?: WorkloadId, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: EntityTypes.Source, target: id ? getSseTargetFromId(id, EntityTypes.Source) : undefined, hideFromHistory });
  };

  const [mutateRestartWorkloads] = useMutation<{ restartWorkloads: boolean }, { sourceIds: WorkloadId[] }>(RESTART_WORKLOADS, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Update, error.cause?.message || error.message),
  });
  const [mutateRestartPod] = useMutation<{ restartPod: boolean }, { namespace: string; name: string }>(RESTART_POD, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Update, error.cause?.message || error.message),
  });
  const [mutateRecoverFromRollback] = useMutation<{ recoverFromRollbackForWorkload: boolean }, { sourceId: WorkloadId }>(RECOVER_FROM_ROLLBACK, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Update, error.cause?.message || error.message),
  });

  const restartWorkloads: UseWorkloadUtils['restartWorkloads'] = async (sourceIds) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      notifyUser(StatusType.Default, 'Pending', 'Restarting sources...', undefined, true);

      const { data } = await mutateRestartWorkloads({ variables: { sourceIds } });
      if (data?.restartWorkloads) notifyUser(StatusType.Success, Crud.Update, `Successfully restarted ${sourceIds.length} sources`);
    }
  };

  const restartPod: UseWorkloadUtils['restartPod'] = async (namespace, name) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      notifyUser(StatusType.Default, 'Pending', 'Restarting pod...', undefined, true);
    }

    const { data } = await mutateRestartPod({ variables: { namespace, name } });
    if (data?.restartPod) notifyUser(StatusType.Success, Crud.Update, `Successfully restarted pod ${namespace}/${name}`);
  };

  const recoverFromRollback: UseWorkloadUtils['recoverFromRollback'] = async (sourceId) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      notifyUser(StatusType.Default, 'Pending', 'Recovering from rollback...', undefined, true);

      const { data } = await mutateRecoverFromRollback({ variables: { sourceId } });
      if (data?.recoverFromRollbackForWorkload) notifyUser(StatusType.Success, Crud.Update, 'Successfully triggered recovery from rollback');
    }
  };

  return {
    restartWorkloads,
    restartPod,
    recoverFromRollback,
  };
};
