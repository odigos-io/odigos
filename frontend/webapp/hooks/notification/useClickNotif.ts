import { useSourceCRUD } from '../sources';
import { useActionCRUD } from '../actions';
import { getIdFromSseTarget } from '@/utils';
import { Types } from '@odigos/ui-components';
import { useDestinationCRUD } from '../destinations';
import { type Notification, WorkloadId } from '@/types';
import { useInstrumentationRuleCRUD } from '../instrumentation-rules';
import { DrawerItem, useDrawerStore, useNotificationStore } from '@/store';

export const useClickNotif = () => {
  const { sources } = useSourceCRUD();
  const { actions } = useActionCRUD();
  const { destinations } = useDestinationCRUD();
  const { instrumentationRules } = useInstrumentationRuleCRUD();
  const { markAsDismissed, markAsSeen } = useNotificationStore();
  const setSelectedItem = useDrawerStore(({ setSelectedItem }) => setSelectedItem);

  const clickNotif = (notif: Notification, options?: { dismissToast?: boolean }) => {
    const { id, crdType, target } = notif;
    const { dismissToast } = options || {};

    if (crdType && target) {
      const drawerItem: Partial<DrawerItem> = {};

      switch (crdType) {
        case Types.ENTITY_TYPES.INSTRUMENTATION_RULE:
          drawerItem['type'] = Types.ENTITY_TYPES.INSTRUMENTATION_RULE;
          drawerItem['id'] = getIdFromSseTarget(target, Types.ENTITY_TYPES.INSTRUMENTATION_RULE);
          drawerItem['item'] = instrumentationRules.find((item) => item.ruleId === drawerItem['id']);
          break;

        case Types.ENTITY_TYPES.SOURCE:
        case 'InstrumentationConfig':
        case 'InstrumentationInstance':
          drawerItem['type'] = Types.ENTITY_TYPES.SOURCE;
          drawerItem['id'] = getIdFromSseTarget(target, Types.ENTITY_TYPES.SOURCE);
          drawerItem['item'] = sources.find(
            (item) => item.kind === (drawerItem['id'] as WorkloadId).kind && item.name === (drawerItem['id'] as WorkloadId).name && item.namespace === (drawerItem['id'] as WorkloadId).namespace,
          );

          break;

        case Types.ENTITY_TYPES.ACTION:
          drawerItem['type'] = Types.ENTITY_TYPES.ACTION;
          drawerItem['id'] = getIdFromSseTarget(target, Types.ENTITY_TYPES.ACTION);
          drawerItem['item'] = actions.find((item) => item.id === drawerItem['id']);
          break;

        case Types.ENTITY_TYPES.DESTINATION:
        case 'Destination':
          drawerItem['type'] = Types.ENTITY_TYPES.DESTINATION;
          drawerItem['id'] = getIdFromSseTarget(target, Types.ENTITY_TYPES.DESTINATION);
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
