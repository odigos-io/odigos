import React from 'react';
import Image from 'next/image';
import { useAppStore } from '@/store';
import styled from 'styled-components';
import { getStatusIcon } from '@/utils';
import { Checkbox, DataTab } from '@/reuseable-components';
import { Handle, type Node, type NodeProps, Position } from '@xyflow/react';
import { type ActionDataParsed, type ActualDestination, type InstrumentationRuleSpec, type K8sActualSource, OVERVIEW_ENTITY_TYPES, STATUSES, WorkloadId } from '@/types';

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
        imageUri: string;
        monitors?: string[];
        isActive?: boolean;
        raw: InstrumentationRuleSpec | K8sActualSource | ActionDataParsed | ActualDestination;
      },
      'base'
    >
  > {}

const Container = styled.div<{ $nodeWidth: Props['data']['nodeWidth'] }>`
  width: ${({ $nodeWidth }) => `${$nodeWidth}px`};
`;

const BaseNode: React.FC<Props> = ({ id: nodeId, data }) => {
  const { nodeWidth, type, status, title, subTitle, imageUri, monitors, isActive, raw } = data;
  const isError = status === STATUSES.UNHEALTHY;

  const { configuredSources, setConfiguredSources } = useAppStore((state) => state);

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
        {/* TODO: handle instrumentation rules for sources */}
        {isError ? (
          <Image src={getStatusIcon('error')} alt='' width={20} height={20} />
        ) : // : type === 'source' && SOME_INDICATOR_THAT_THIS_IS_INSTRUMENTED ? ( <Image src={getEntityIcon(OVERVIEW_ENTITY_TYPES.RULE)} alt='' width={18} height={18} /> )
        null}

        {type === 'source' ? <Checkbox initialValue={getSourceLocation().index !== -1} onChange={onSelectSource} /> : null}
      </>
    );
  };

  const renderHandles = () => {
    switch (type) {
      case 'source':
        return <Handle type='source' position={Position.Right} style={{ visibility: 'hidden' }} />;
      case 'destination':
        return <Handle type='target' position={Position.Left} style={{ visibility: 'hidden' }} />;
      default:
        return null;
    }
  };

  return (
    <Container data-id={nodeId} $nodeWidth={nodeWidth} className='nowheel nodrag'>
      <DataTab title={title} subTitle={subTitle} logo={imageUri} monitors={monitors} isActive={isActive} isError={isError} onClick={() => {}}>
        {renderActions()}
        {renderHandles()}
      </DataTab>
    </Container>
  );
};

export default BaseNode;
