import React from 'react';
import { Handle, Position } from '@xyflow/react';

const GroupNode = () => {
  return (
    <>
      <Handle type='source' position={Position.Right} id='group-output' isConnectable style={{ visibility: 'hidden' }} />
      <Handle type='target' position={Position.Left} id='group-input' isConnectable style={{ visibility: 'hidden' }} />
    </>
  );
};

export default GroupNode;
