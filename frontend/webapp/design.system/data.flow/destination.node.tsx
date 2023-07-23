import React, { memo } from "react";
import { Handle, Position } from "reactflow";
import Logo from "assets/logos/code-sandbox-logo.svg";
import { styled } from "styled-components";
import { KeyvalText } from "@/design.system";

const NamespaceContainer = styled.div`
  display: flex;
  padding: 8px;
  border-radius: 8px;
  border: 1px solid var(--dark-mode-odigos-torquiz, #96f3ff8e);
  background: var(--dark-mode-dark-1, #0a1824);
  align-items: center;
  min-width: 200px;
  gap: 6px;
`;

const TextWrapper = styled.div`
  gap: 10px;
`;

export default memo(({ data, isConnectable }: any) => {
  return (
    <NamespaceContainer>
      <Logo width={40} />
      <TextWrapper>
        <KeyvalText size={14} weight={600}>
          {"kube-system"}
        </KeyvalText>
        <KeyvalText color={"#8b92a5"}>{"80 apps"}</KeyvalText>
      </TextWrapper>
      <Handle
        type="target"
        position={Position.Left}
        id="a"
        isConnectable={isConnectable}
        style={{ visibility: "hidden" }}
      />
    </NamespaceContainer>
  );
});
