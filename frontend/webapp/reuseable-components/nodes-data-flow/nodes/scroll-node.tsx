import React, { useEffect, useRef } from 'react';
import BaseNode from './base-node';
import styled from 'styled-components';
import { type Node, type NodeProps } from '@xyflow/react';
import { type K8sActualSource, OVERVIEW_ENTITY_TYPES, STATUSES, type WorkloadId } from '@/types';
import { useDrawerStore } from '@/store';

interface Props
  extends NodeProps<
    Node<
      {
        nodeWidth: number;
        nodeHeight: number;
        items: NodeProps<
          Node<
            {
              nodeWidth: number;
              framePadding: number;
              id: WorkloadId;
              type: OVERVIEW_ENTITY_TYPES;
              status: STATUSES;
              title: string;
              subTitle: string;
              imageUri: string;
              raw: K8sActualSource;
            },
            'scroll-item'
          >
        >[];
        onScroll: (params: { clientHeight: number; scrollHeight: number; scrollTop: number }) => void;
      },
      'scroll'
    >
  > {}

const Container = styled.div<{ $nodeWidth: number; $nodeHeight: number }>`
  width: ${({ $nodeWidth }) => $nodeWidth}px;
  height: ${({ $nodeHeight }) => $nodeHeight}px;
  background: transparent;
  border: none;
  overflow-y: auto;
`;

const BaseNodeWrapper = styled.div<{ $framePadding: number }>`
  margin: ${({ $framePadding }) => $framePadding}px 0;
`;

const ScrollNode: React.FC<Props> = ({ data, ...rest }) => {
  const { nodeWidth, nodeHeight, items, onScroll } = data;

  const { setSelectedItem } = useDrawerStore();
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleScroll = (e: Event) => {
      e.stopPropagation();

      // @ts-ignore - these properties are available on the Event, TS is not aware of it
      const { clientHeight, scrollHeight, scrollTop } = e.target || { clientHeight: 0, scrollHeight: 0, scrollTop: 0 };
      const isTop = scrollTop === 0;
      const isBottom = scrollHeight - scrollTop <= clientHeight;

      if (isTop) {
        console.log('Reached top of scroll-node');
      } else if (isBottom) {
        console.log('Reached bottom of scroll-node');
      }

      onScroll({ clientHeight, scrollHeight, scrollTop });
    };

    const { current } = containerRef;

    current?.addEventListener('scroll', handleScroll);
    return () => current?.removeEventListener('scroll', handleScroll);
  }, [onScroll]);

  return (
    <Container ref={containerRef} $nodeWidth={nodeWidth} $nodeHeight={nodeHeight} className='nowheel nodrag'>
      {items.map((item) => (
        <BaseNodeWrapper
          key={item.id}
          $framePadding={item.data.framePadding}
          onClick={(e) => {
            e.stopPropagation();
            setSelectedItem({ id: item.id, type: item.data.type, item: item.data.raw });
          }}
        >
          <BaseNode {...rest} type='base' id={item.id} data={item.data} />
        </BaseNodeWrapper>
      ))}
    </Container>
  );
};

export default ScrollNode;
