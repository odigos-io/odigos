// src/hooks/useNodeDataFlowHandlers.ts
import { useCallback } from 'react';
import { useDrawerStore } from '@/store';
import { K8sActualSource, ActualDestination, ActionDataParsed, OVERVIEW_ENTITY_TYPES } from '@/types';

export function useNodeDataFlowHandlers({
  sources,
  actions,
  destinations,
}: {
  sources: K8sActualSource[];
  actions: ActionDataParsed[];
  destinations: ActualDestination[];
}) {
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
      }

      if (type === OVERVIEW_ENTITY_TYPES.ACTION) {
        const selectedDrawerItem = actions.find((action) => action.id === id);
        if (!selectedDrawerItem) return;

        setSelectedItem({
          id,
          type,
          item: selectedDrawerItem,
        });
      }

      if (type === OVERVIEW_ENTITY_TYPES.DESTINATION) {
        const selectedDrawerItem = destinations.find((destination) => destination.id === id);
        if (!selectedDrawerItem) return;

        setSelectedItem({
          id,
          type,
          item: selectedDrawerItem,
        });
      }
    },
    [sources, actions, destinations, setSelectedItem]
  );

  return {
    handleNodeClick,
  };
}
