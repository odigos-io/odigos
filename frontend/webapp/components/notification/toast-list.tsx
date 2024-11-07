import React from 'react';
import styled from 'styled-components';
import { NotificationNote } from '@/reuseable-components';
import { Notification, OVERVIEW_ENTITY_TYPES } from '@/types';
import { DrawerBaseItem, useDrawerStore, useNotificationStore } from '@/store';
import { useActualDestination, useActualSources, useGetActions, useGetInstrumentationRules } from '@/hooks';
import { getIdFromSseTarget } from '@/utils';

const Container = styled.div`
  position: fixed;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 10000;
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 600px;
`;

export const ToastList: React.FC = () => {
  const { notifications } = useNotificationStore();

  return (
    <Container>
      {notifications
        .filter(({ dismissed }) => !dismissed)
        .map((notif) => (
          <Toast key={`toast-${notif.id}`} {...notif} />
        ))}
    </Container>
  );
};

const Toast: React.FC<Notification> = ({ id, type, title, message, crdType, target }) => {
  const { markAsDismissed, markAsSeen } = useNotificationStore();

  const { actions } = useGetActions();
  const { sources } = useActualSources();
  const { destinations } = useActualDestination();
  const { instrumentationRules } = useGetInstrumentationRules();
  const setSelectedItem = useDrawerStore(({ setSelectedItem }) => setSelectedItem);

  const onClick = () => {
    if (crdType && target) {
      const drawerItem: Partial<DrawerBaseItem> = {};

      switch (crdType) {
        case OVERVIEW_ENTITY_TYPES.RULE:
          drawerItem['type'] = OVERVIEW_ENTITY_TYPES.RULE;
          drawerItem['id'] = getIdFromSseTarget(target, OVERVIEW_ENTITY_TYPES.RULE);
          drawerItem['item'] = instrumentationRules.find((item) => item.ruleId === drawerItem['id']);
          break;

        case OVERVIEW_ENTITY_TYPES.SOURCE:
        case 'InstrumentedApplication':
        case 'InstrumentationInstance':
          drawerItem['type'] = OVERVIEW_ENTITY_TYPES.SOURCE;
          drawerItem['id'] = getIdFromSseTarget(target, OVERVIEW_ENTITY_TYPES.SOURCE);
          drawerItem['item'] = sources.find(
            (item) =>
              item.kind === drawerItem['id']?.['kind'] &&
              item.name === drawerItem['id']?.['name'] &&
              item.namespace === drawerItem['id']?.['namespace']
          );
          break;

        case OVERVIEW_ENTITY_TYPES.ACTION:
          drawerItem['type'] = OVERVIEW_ENTITY_TYPES.ACTION;
          drawerItem['id'] = getIdFromSseTarget(target, OVERVIEW_ENTITY_TYPES.ACTION);
          drawerItem['item'] = actions.find((item) => item.id === drawerItem['id']);
          break;

        case OVERVIEW_ENTITY_TYPES.DESTINATION:
        case 'Destination':
          drawerItem['type'] = OVERVIEW_ENTITY_TYPES.DESTINATION;
          drawerItem['id'] = getIdFromSseTarget(target, OVERVIEW_ENTITY_TYPES.DESTINATION);
          drawerItem['item'] = destinations.find((item) => item.id === drawerItem['id']);
          break;

        default:
          break;
      }

      if (!!drawerItem.item) {
        setSelectedItem(drawerItem as DrawerBaseItem);
      }
    }

    markAsSeen(id);
    markAsDismissed(id);
  };

  const onClose = ({ asSeen }) => {
    markAsDismissed(id);
    if (asSeen) markAsSeen(id);
  };

  return (
    <NotificationNote
      id={id}
      type={type}
      title={title}
      message={message}
      action={
        crdType && target
          ? {
              label: 'go to details',
              onClick,
            }
          : undefined
      }
      onClose={onClose}
    />
  );
};
