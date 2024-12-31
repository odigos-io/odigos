import React from 'react';
import styled from 'styled-components';
import { ErrorTriangleIcon, SVG } from '@/assets';
import { useAppStore, usePendingStore } from '@/store';
import { Checkbox, DataTab, FadeLoader } from '@/reuseable-components';
import { Handle, type Node, type NodeProps, Position } from '@xyflow/react';
import { type ActionDataParsed, type ActualDestination, type InstrumentationRuleSpec, type K8sActualSource, NODE_TYPES, OVERVIEW_ENTITY_TYPES, STATUSES, WorkloadId } from '@/types';

interface Props
  extends NodeProps<
    Node<
      {
        nodeWidth: number;
        id: string | WorkloadId;
        type: OVERVIEW_ENTITY_TYPES;
        status: STATUSES;
        title: string;
        subTitle: string;
        icon?: SVG;
        iconSrc?: string;
        monitors?: string[];
        isActive?: boolean;
        raw: InstrumentationRuleSpec | K8sActualSource | ActionDataParsed | ActualDestination;
      },
      NODE_TYPES.BASE
    >
  > {}

const Container = styled.div<{ $nodeWidth: Props['data']['nodeWidth'] }>`
  width: ${({ $nodeWidth }) => `${$nodeWidth}px`};
`;

const BaseNode: React.FC<Props> = ({ id: nodeId, data }) => {
  const { nodeWidth, id: entityId, type: entityType, status, title, subTitle, icon, iconSrc, monitors, isActive, raw } = data;
  const isError = status === STATUSES.UNHEALTHY;

  const { configuredSources, setConfiguredSources } = useAppStore();
  const { isThisPending } = usePendingStore();
  const isPending = isThisPending({ entityType, entityId });

  const renderActions = () => {
    const getSourceLocation = () => {
      const { namespace, name, kind } = raw as K8sActualSource;
      const selected = { ...configuredSources };
      if (!selected[namespace]) selected[namespace] = [];

      const index = selected[namespace].findIndex((x) => x.name === name && x.kind === kind);
      return { index, namespace, selected };
    };

    const onSelectSource = () => {
      const { index, namespace, selected } = getSourceLocation();

      if (index === -1) {
        selected[namespace].push(raw as K8sActualSource);
      } else {
        selected[namespace].splice(index, 1);
      }

      setConfiguredSources(selected);
    };

    return (
      <>
        {/* TODO: handle action/icon to apply instrumentation-rules for individual sources (@Notion GEN-1650) */}
        {isPending ? <FadeLoader /> : isError ? <ErrorTriangleIcon size={20} /> : null}
        {entityType === 'source' ? <Checkbox value={getSourceLocation().index !== -1} onChange={onSelectSource} disabled={isPending} /> : null}
      </>
    );
  };

  return (
    <Container data-id={nodeId} $nodeWidth={nodeWidth} className='nowheel nodrag'>
      <DataTab title={title} subTitle={subTitle} icon={icon} iconSrc={iconSrc} monitors={monitors} isActive={isActive} isError={isError} onClick={() => {}} renderActions={renderActions} />
      <Handle type='target' position={Position.Left} style={{ visibility: 'hidden' }} />
      <Handle type='source' position={Position.Right} style={{ visibility: 'hidden' }} />
    </Container>
  );
};

export default BaseNode;
