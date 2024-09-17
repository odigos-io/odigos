'use client';
import React, { useMemo } from 'react';
import { ReactFlow } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import BaseNode, { NodeDataProps } from './nodes/base-node';
import { useActualDestination, useActualSources, useGetActions } from '@/hooks';
import { getMainContainerLanguageLogo } from '@/utils/constants/programming-languages';
import Image from 'next/image';
import { Text } from '@/reuseable-components';
import theme from '@/styles/theme';
import { DataFlowHeader } from './header';

const nodeTypes = {
  base: BaseNode,
};

const initialEdges = [{ id: 'e1-2', source: '1', target: '2' }];

export function NodeBaseDataFlow() {
  const { actions } = useGetActions();
  const { sources } = useActualSources();
  const { destinations } = useActualDestination();

  console.log({ destinations });

  const sourcesNode = useMemo(() => {
    return sources.map((source, index) => ({
      type: 'base',
      id: `source-${index}`,
      position: { x: 0, y: 100 * (index + 1) },
      data: {
        type: 'source',
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

  const destinationNode = useMemo(() => {
    return destinations.map((destination, index) => ({
      type: 'base',
      id: `destination-${index}`,
      position: { x: 1000, y: 100 * (index + 1) },
      data: {
        type: 'destination',
        title: destination.destinationType.displayName,
        subTitle: 'Destination',
        imageUri: destination.destinationType.imageUrl,
        monitors: destination.exportedSignals,
        status: 'healthy',
        onClick: () => {
          console.log(destination);
        },
      },
    }));
  }, [destinations]);

  const actionsNode = useMemo(() => {
    return actions.map((action, index) => ({
      type: 'base',
      id: `action-${index}`,
      position: { x: 500, y: 100 * (index + 1) },
      data: {
        type: 'action',
        title: action.type,
        subTitle: 'Action',
        imageUri: '/icons/common/action.svg',
        status: 'healthy',
        onClick: () => {
          console.log(action);
        },
      },
    }));
  }, [actions]);

  const COLUMNS = [
    {
      icon: '/icons/overview/sources.svg',
      title: 'Sources',
      tagValue: sources.length,
    },
    {
      icon: '/icons/overview/actions.svg',
      title: 'Actions',
      tagValue: actions.length,
    },
    {
      icon: '/icons/overview/destinations.svg',
      title: 'Destinations',
      tagValue: destinations.length,
    },
  ];

  return (
    <div
      style={{ height: '100vh', padding: '0 32px', width: 'calc(100% - 64px)' }}
    >
      <DataFlowHeader columns={COLUMNS} />
      <ReactFlow
        nodeTypes={nodeTypes}
        nodes={[...sourcesNode, ...destinationNode, ...actionsNode]}
        edges={initialEdges}
      />
    </div>
  );
}
