// src/hooks/useNodeDataFlowHandlers.ts
import { useCallback } from 'react';
import { useDrawerStore, useModalStore } from '@/store';
import { K8sActualSource, ActualDestination, ActionDataParsed, OVERVIEW_ENTITY_TYPES, OVERVIEW_NODE_TYPES } from '@/types';

export function useNodeDataFlowHandlers({
  rules,
  sources,
  actions,
  destinations,
}: {
  rules: any[];
  sources: K8sActualSource[];
  actions: ActionDataParsed[];
  destinations: ActualDestination[];
}) {
  const { setCurrentModal } = useModalStore();
  const setSelectedItem = useDrawerStore(({ setSelectedItem }) => setSelectedItem);

  const handleNodeClick = useCallback(
    (_, object: any) => {
      const {
        data: { id, type },
      } = object;

      if (type === OVERVIEW_ENTITY_TYPES.SOURCE) {
        const selectedDrawerItem = sources.find(({ kind, name, namespace }) => kind === id.kind && name === id.name && namespace === id.namespace);
        if (!selectedDrawerItem) return;

        const { kind, name, namespace } = selectedDrawerItem;

        setSelectedItem({
          id: { kind, name, namespace },
          type,
          item: selectedDrawerItem,
        });
      } else if (type === OVERVIEW_ENTITY_TYPES.ACTION) {
        const selectedDrawerItem = actions.find((action) => action.id === id);
        if (!selectedDrawerItem) return;

        setSelectedItem({
          id,
          type,
          item: selectedDrawerItem,
        });
      } else if (type === OVERVIEW_ENTITY_TYPES.DESTINATION) {
        const selectedDrawerItem = destinations.find((destination) => destination.id === id);
        if (!selectedDrawerItem) return;

        setSelectedItem({
          id,
          type,
          item: selectedDrawerItem,
        });
      } else if (type === OVERVIEW_NODE_TYPES.ADD_RULE) {
        setCurrentModal(OVERVIEW_ENTITY_TYPES.RULE);
      } else if (type === OVERVIEW_NODE_TYPES.ADD_SOURCE) {
        setCurrentModal(OVERVIEW_ENTITY_TYPES.SOURCE);
      } else if (type === OVERVIEW_NODE_TYPES.ADD_ACTION) {
        setCurrentModal(OVERVIEW_ENTITY_TYPES.ACTION);
      } else if (type === OVERVIEW_NODE_TYPES.ADD_DESTIONATION) {
        setCurrentModal(OVERVIEW_ENTITY_TYPES.DESTINATION);
      }
    },
    [sources, actions, destinations, setSelectedItem]
  );

  return {
    handleNodeClick,
  };
}
