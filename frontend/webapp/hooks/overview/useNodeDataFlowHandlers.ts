// src/hooks/useNodeDataFlowHandlers.ts
import { useCallback } from 'react';
import { useDrawerStore } from '@/store';
import { K8sActualSource, ActualDestination } from '@/types';

const TYPE_SOURCE = 'source';
const TYPE_DESTINATION = 'destination';

export function useNodeDataFlowHandlers(
  sources: K8sActualSource[],
  destinations: ActualDestination[]
) {
  const setSelectedItem = useDrawerStore(
    ({ setSelectedItem }) => setSelectedItem
  );

  const handleNodeClick = useCallback(
    (_, object: any) => {
      if (object.data.type === TYPE_SOURCE) {
        const { id } = object.data;
        const selectedDrawerItem = sources.find(
          ({ kind, name, namespace }) =>
            kind === id.kind && name === id.name && namespace === id.namespace
        );
        if (!selectedDrawerItem) return;

        const { kind, name, namespace } = selectedDrawerItem;

        setSelectedItem({
          id: { kind, name, namespace },
          item: selectedDrawerItem,
          type: TYPE_SOURCE,
        });
      }

      if (object.data.type === TYPE_DESTINATION) {
        const { id } = object.data;
        const selectedDrawerItem = destinations.find(
          (destination) => destination.id === id
        );

        setSelectedItem({
          id,
          item: selectedDrawerItem,
          type: TYPE_DESTINATION,
        });
      }
    },
    [sources, destinations, setSelectedItem]
  );

  return {
    handleNodeClick,
  };
}
