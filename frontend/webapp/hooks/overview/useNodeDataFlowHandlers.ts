// src/hooks/useNodeDataFlowHandlers.ts
import { useCallback } from 'react';
import { useSourceCRUD } from '../sources';
import { useActionCRUD } from '../actions';
import { useDestinationCRUD } from '../destinations';
import { useDrawerStore, useModalStore } from '@/store';
import { OVERVIEW_ENTITY_TYPES, OVERVIEW_NODE_TYPES, WorkloadId } from '@/types';
import { useInstrumentationRuleCRUD } from '../instrumentation-rules';
import { Node } from '@xyflow/react';

export function useNodeDataFlowHandlers() {
  const { sources } = useSourceCRUD();
  const { actions } = useActionCRUD();
  const { destinations } = useDestinationCRUD();
  const { instrumentationRules } = useInstrumentationRuleCRUD();

  const { setCurrentModal } = useModalStore();
  const setSelectedItem = useDrawerStore(({ setSelectedItem }) => setSelectedItem);

  const handleNodeClick = useCallback(
    (
      _: React.MouseEvent,
      object: Node<
        {
          id: string | WorkloadId;
          type: OVERVIEW_ENTITY_TYPES | OVERVIEW_NODE_TYPES;
        },
        'add'
      >,
    ) => {
      const {
        data: { id, type },
      } = object;

      const entities = {
        sources,
        actions,
        destinations,
        rules: instrumentationRules,
      };

      if (type === OVERVIEW_ENTITY_TYPES.SOURCE) {
        const { kind, name, namespace } = id as WorkloadId;
        const selectedDrawerItem = entities['sources'].find((item) => item.kind === kind && item.name === name && item.namespace === namespace);
        if (!selectedDrawerItem) return;

        setSelectedItem({
          id,
          type,
          item: selectedDrawerItem,
        });
      } else if ([OVERVIEW_ENTITY_TYPES.RULE, OVERVIEW_ENTITY_TYPES.ACTION, OVERVIEW_ENTITY_TYPES.DESTINATION].includes(type as OVERVIEW_ENTITY_TYPES)) {
        const selectedDrawerItem = entities[`${type}s`].find((item) => id && [item.id, item.ruleId].includes(id));
        if (!selectedDrawerItem) return;

        setSelectedItem({
          id,
          type: type as OVERVIEW_ENTITY_TYPES,
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
    [sources, actions, destinations, instrumentationRules, setSelectedItem, setCurrentModal],
  );

  return {
    handleNodeClick,
  };
}
