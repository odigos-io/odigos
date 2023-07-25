import React, { memo } from "react";
import { Handle, Position } from "reactflow";
import { styled } from "styled-components";
import { KeyvalText } from "@/design.system";
import { Folder } from "@/assets/icons/overview";

const NamespaceContainer = styled.div`
  display: flex;
  padding: 16px;
  border-radius: 12px;
  border: ${({ theme }) => `solid 1px ${theme.colors.blue_grey}`};
  background: ${({ theme }) => theme.colors.light_dark};
  align-items: center;
  width: 272px;
  gap: 8px;
`;

const TextWrapper = styled.div`
  gap: 10px;
`;

export default memo(({ data, isConnectable }: any) => {
  return (
    <NamespaceContainer>
      <Folder width={32} />
      <TextWrapper>
        <KeyvalText size={14} weight={600}>
          {data?.name}
        </KeyvalText>
        <KeyvalText
          color={"#8b92a5"}
        >{`${data?.totalAppsInstrumented} Apps Instrumented`}</KeyvalText>
      </TextWrapper>
      <Handle
        type="source"
        position={Position.Right}
        id="a"
        isConnectable={isConnectable}
        style={{ visibility: "hidden" }}
      />
    </NamespaceContainer>
  );
});
