import { useSourceCRUD } from '../sources';
import { useActionCRUD } from '../actions';
import { type Notification } from '@/types';
import { useDestinationCRUD } from '../destinations';
import { useInstrumentationRuleCRUD } from '../instrumentation-rules';
import { ENTITY_TYPES, getIdFromSseTarget, type WorkloadId } from '@odigos/ui-utils';
import { DrawerItem, useDrawerStore, useNotificationStore } from '@odigos/ui-containers';

export const useClickNotif = () => {
  const { sources } = useSourceCRUD();
  const { actions } = useActionCRUD();
  const { destinations } = useDestinationCRUD();
  const { instrumentationRules } = useInstrumentationRuleCRUD();

  const { setSelectedItem } = useDrawerStore();
  const { markAsDismissed, markAsSeen } = useNotificationStore();

  const clickNotif = (notif: Notification, options?: { dismissToast?: boolean }) => {
    const { id, crdType, target } = notif;
    const { dismissToast } = options || {};

    if (crdType && target) {
      const drawerItem: Partial<DrawerItem> = {};

      switch (crdType) {
        case ENTITY_TYPES.INSTRUMENTATION_RULE:
          drawerItem['type'] = ENTITY_TYPES.INSTRUMENTATION_RULE;
          drawerItem['id'] = getIdFromSseTarget(target, ENTITY_TYPES.INSTRUMENTATION_RULE);
          drawerItem['item'] = instrumentationRules.find((item) => item.ruleId === drawerItem['id']);
          break;

        case ENTITY_TYPES.SOURCE:
        case 'InstrumentationConfig':
        case 'InstrumentationInstance':
          drawerItem['type'] = ENTITY_TYPES.SOURCE;
          drawerItem['id'] = getIdFromSseTarget(target, ENTITY_TYPES.SOURCE);
          drawerItem['item'] = sources.find(
            (item) => item.kind === (drawerItem['id'] as WorkloadId).kind && item.name === (drawerItem['id'] as WorkloadId).name && item.namespace === (drawerItem['id'] as WorkloadId).namespace,
          );

          break;

        case ENTITY_TYPES.ACTION:
          drawerItem['type'] = ENTITY_TYPES.ACTION;
          drawerItem['id'] = getIdFromSseTarget(target, ENTITY_TYPES.ACTION);
          drawerItem['item'] = actions.find((item) => item.id === drawerItem['id']);
          break;

        case ENTITY_TYPES.DESTINATION:
        case 'Destination':
          drawerItem['type'] = ENTITY_TYPES.DESTINATION;
          drawerItem['id'] = getIdFromSseTarget(target, ENTITY_TYPES.DESTINATION);
          drawerItem['item'] = destinations.find((item) => item.id === drawerItem['id']);
          break;

        default:
          console.warn('notif click not handled for:', { crdType, target });
          break;
      }

      if (!!drawerItem.item) {
        setSelectedItem(drawerItem as DrawerItem);
      } else {
        console.warn('notif item not found for:', { crdType, target });
      }
    }

    markAsSeen(id);
    if (dismissToast) markAsDismissed(id);
  };

  return clickNotif;
};
