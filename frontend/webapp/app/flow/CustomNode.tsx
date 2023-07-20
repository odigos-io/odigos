import React, { memo } from "react";
import { Handle, Position } from "reactflow";
import Connect from "assets/icons/connect.svg";
export default memo(({ data, isConnectable }: any) => {
  return (
    <div>
      <Handle
        type="target"
        position={Position.Left}
        onConnect={(params) => console.log("handle onConnect", params)}
        style={{ visibility: "hidden" }}
      />
      <Connect />

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
