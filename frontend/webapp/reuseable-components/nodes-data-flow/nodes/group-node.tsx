import React from 'react';
import { Handle, type Node, type NodeProps, Position } from '@xyflow/react';

interface Props extends NodeProps<Node<{}, 'group'>> {}

const GroupNode: React.FC<Props> = () => {
  return (
    <>
      <Handle type='source' position={Position.Right} id='group-output' isConnectable style={{ visibility: 'hidden' }} />
      <Handle type='target' position={Position.Left} id='group-input' isConnectable style={{ visibility: 'hidden' }} />
    </>
  );
};

export default GroupNode;
