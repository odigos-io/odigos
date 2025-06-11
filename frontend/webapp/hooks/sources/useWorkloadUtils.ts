import { useConfig } from '../config';
import { useMutation } from '@apollo/client';
import { RESTART_WORKLOADS } from '@/graphql';
import { getSseTargetFromId } from '@odigos/ui-kit/functions';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { useNotificationStore, usePendingStore } from '@odigos/ui-kit/store';
import { type WorkloadId, EntityTypes, StatusType, Crud } from '@odigos/ui-kit/types';

interface UseWorkloadUtils {
  restartWorkloads: (sourceIds: WorkloadId[]) => Promise<void>;
}

export const useWorkloadUtils = (): UseWorkloadUtils => {
  const { isReadonly } = useConfig();
  const { addNotification } = useNotificationStore();
  const { addPendingItems, removePendingItems } = usePendingStore();

  const notifyUser = (type: StatusType, title: string, message: string, id?: WorkloadId, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, crdType: EntityTypes.Source, target: id ? getSseTargetFromId(id, EntityTypes.Source) : undefined, hideFromHistory });
  };

  const [mutateRestartWorkloads] = useMutation<{ restartWorkloads: boolean }, { sourceIds: WorkloadId[] }>(RESTART_WORKLOADS, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Update, error.cause?.message || error.message),
  });

  const restartWorkloads: UseWorkloadUtils['restartWorkloads'] = async (sourceIds) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      notifyUser(StatusType.Default, 'Pending', 'Restarting sources...', undefined, true);

      const pendingItems = sourceIds.map((sourceId) => ({ entityType: EntityTypes.Source, entityId: sourceId }));
      addPendingItems(pendingItems);

      const { errors } = await mutateRestartWorkloads({ variables: { sourceIds } });

      if (!errors?.length) notifyUser(StatusType.Success, Crud.Update, `Successfully restarted ${sourceIds.length} sources`);
      removePendingItems(pendingItems);
    }
  };

  return {
    restartWorkloads,
  };
};
