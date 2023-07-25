import React, { memo } from "react";
import { Handle, Position } from "reactflow";
import { KeyvalMiddleware } from "@/assets/icons/overview";

export default memo(({ isConnectable }: any) => {
  return (
    <div>
      <Handle
        type="target"
        position={Position.Left}
        style={{ visibility: "hidden" }}
      />
      <KeyvalMiddleware />
      <Handle
        type="source"
        position={Position.Right}
        id="a"
        isConnectable={isConnectable}
        style={{ visibility: "hidden" }}
      />
    </div>
  );
});
