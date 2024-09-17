'use client';
import React, { useMemo } from 'react';
import { ReactFlow } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import BaseNode, { NodeDataProps } from './nodes/base-node';
import { useActualDestination, useActualSources, useGetActions } from '@/hooks';
import { getMainContainerLanguageLogo } from '@/utils/constants/programming-languages';

const nodeTypes = {
  base: BaseNode,
};

const initialEdges = [{ id: 'e1-2', source: '1', target: '2' }];

export function NodeBaseDataFlow() {
  const { actions } = useGetActions();
  const { sources } = useActualSources();
  const { destinations } = useActualDestination();

  const sourcesNode = useMemo(() => {
    return sources.map((source, index) => ({
      type: 'base',
      id: `source-${index}`,
      position: { x: 0, y: 100 * (index + 1) },
      data: {
        title: source.name,
        subTitle: source.kind,
        imageUri: getMainContainerLanguageLogo(source),
        status: 'healthy',
        onClick: () => {
          console.log(source);
        },
      },
    }));
  }, [sources]);
  return (
    <div style={{ width: '100vw', height: '100vh' }}>
      <ReactFlow
        nodeTypes={nodeTypes}
        nodes={sourcesNode}
        edges={initialEdges}
      />
    </div>
  );
}
