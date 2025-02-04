import { useCallback } from 'react';
import { Node } from '@xyflow/react';
import { useSourceCRUD } from '../sources';
import { useActionCRUD } from '../actions';
import { OVERVIEW_NODE_TYPES } from '@/types';
import { useDestinationCRUD } from '../destinations';
import { useDrawerStore, useModalStore } from '@/store';
import { ENTITY_TYPES, type WorkloadId } from '@odigos/ui-utils';
import { useInstrumentationRuleCRUD } from '../instrumentation-rules';

export const useNodeDataFlowHandlers = () => {
  const { sources } = useSourceCRUD();
  const { actions } = useActionCRUD();
  const { destinations } = useDestinationCRUD();
  const { instrumentationRules } = useInstrumentationRuleCRUD();

  const { setCurrentModal } = useModalStore();
  const { setSelectedItem } = useDrawerStore();

  const handleNodeClick = useCallback(
    (
      _: React.MouseEvent | null,
      object: Node<
        {
          id: string | WorkloadId;
          type: ENTITY_TYPES | OVERVIEW_NODE_TYPES;
        },
        'any-node'
      >,
    ) => {
      const {
        data: { id, type },
      } = object;

      if (type === ENTITY_TYPES.SOURCE) {
        const { kind, name, namespace } = id as WorkloadId;

        const selectedDrawerItem = sources.find((item) => item.kind === kind && item.name === name && item.namespace === namespace);
        if (!selectedDrawerItem) {
          console.warn('Selected item not found', { id, sourcesCount: sources.length });
          return;
        }

        setSelectedItem({
          id,
          type,
          item: selectedDrawerItem,
        });
      } else if (type === ENTITY_TYPES.ACTION) {
        const selectedDrawerItem = actions.find((item) => item.id === id);
        if (!selectedDrawerItem) {
          console.warn('Selected item not found', { id, actionsCount: actions.length });
          return;
        }

        setSelectedItem({
          id,
          type,
          item: selectedDrawerItem,
        });
      } else if (type === ENTITY_TYPES.DESTINATION) {
        const selectedDrawerItem = destinations.find((item) => item.id === id);
        if (!selectedDrawerItem) {
          console.warn('Selected item not found', { id, destinationsCount: destinations.length });
          return;
        }

        setSelectedItem({
          id,
          type,
          item: selectedDrawerItem,
        });
      } else if (type === ENTITY_TYPES.INSTRUMENTATION_RULE) {
        const selectedDrawerItem = instrumentationRules.find((item) => item.ruleId === id);
        if (!selectedDrawerItem) {
          console.warn('Selected item not found', { id, rulesCount: instrumentationRules.length });
          return;
        }

        setSelectedItem({
          id,
          type,
          item: selectedDrawerItem,
        });
      } else if (type === OVERVIEW_NODE_TYPES.ADD_RULE) {
        setCurrentModal(ENTITY_TYPES.INSTRUMENTATION_RULE);
      } else if (type === OVERVIEW_NODE_TYPES.ADD_SOURCE) {
        setCurrentModal(ENTITY_TYPES.SOURCE);
      } else if (type === OVERVIEW_NODE_TYPES.ADD_ACTION) {
        setCurrentModal(ENTITY_TYPES.ACTION);
      } else if (type === OVERVIEW_NODE_TYPES.ADD_DESTINATION) {
        setCurrentModal(ENTITY_TYPES.DESTINATION);
      } else {
        console.warn('Unhandled node click', object);
      }
    },
    [sources, actions, destinations, instrumentationRules, setSelectedItem, setCurrentModal],
  );

  return {
    handleNodeClick,
  };
};
