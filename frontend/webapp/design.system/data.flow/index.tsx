"use client";
import React, { useEffect } from "react";
import ReactFlow, {
  Background,
  Controls,
  useReactFlow,
  ReactFlowProvider,
} from "reactflow";
import CenterNode from "./keyval.middleware";
import NamespaceNode from "./namespace.node";
import DestinationNode from "./destination.node";
import "reactflow/dist/style.css";
import { ControllerWrapper, DataFlowContainer } from "./data.flow.styled";
import { IDataFlow } from "./types";

const backgroundColor = "#132330";

const nodeTypes = {
  custom: CenterNode,
  namespace: NamespaceNode,
  destination: DestinationNode,
};

function DataFlow({ nodes, edges }: IDataFlow) {
  const { fitView } = useReactFlow();

  useEffect(() => {
    setTimeout(() => {
      fitView();
    }, 100);
  }, [fitView, nodes, edges]);

  return (
    <DataFlowContainer>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        nodesDraggable={false}
        nodeOrigin={[0.4, 0.4]}
      >
        <ControllerWrapper>
          <Controls position="top-left" showInteractive={false} />
        </ControllerWrapper>
        <Background gap={12} size={1} style={{ backgroundColor }} />
      </ReactFlow>
    </DataFlowContainer>
  );
}

export function KeyvalFlow(props) {
  return (
    <ReactFlowProvider>
      <DataFlow {...props} />
    </ReactFlowProvider>
  );
}
