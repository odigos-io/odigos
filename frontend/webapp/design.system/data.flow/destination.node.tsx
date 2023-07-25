import React, { memo } from "react";
import { Handle, Position } from "reactflow";
import { styled } from "styled-components";
import { KeyvalImage, KeyvalText } from "@/design.system";
import { MONITORING_OPTIONS } from "@/components/setup/destination/utils";

interface IconWrapperProps {
  tapped?: any;
}

const DestinationNodeContainer = styled.div`
  padding: 16px 24px;
  display: flex;
  border-radius: 12px;
  border: ${({ theme }) => `solid 1px ${theme.colors.blue_grey}`};
  background: ${({ theme }) => theme.colors.light_dark};
  align-items: center;
  justify-content: space-between;
  width: 368px;
`;

export const NodeDataWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;

const TextWrapper = styled.div`
  gap: 8px;
  display: flex;
  flex-direction: column;
`;

const IMAGE_STYLE: React.CSSProperties = {
  backgroundColor: "#fff",
  padding: 4,
};

const IconWrapper = styled.div<IconWrapperProps>`
  padding: 4px;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 16px;
  background: ${({ theme, tapped }) =>
    tapped ? theme.colors.dark_blue : "#0e1c28"};
`;

const MonitorsListWrapper = styled.div`
  display: flex;
  gap: 8px;
`;

export default memo(({ data, isConnectable }: any) => {
  function renderMonitors() {
    return MONITORING_OPTIONS.map((monitor) => (
      <IconWrapper
        key={monitor?.id}
        tapped={data?.signals[monitor?.type] || false}
      >
        {data?.signals[monitor?.type]
          ? monitor.icons.focus()
          : monitor.icons.notFocus()}
      </IconWrapper>
    ));
  }

  return (
    <DestinationNodeContainer>
      <NodeDataWrapper>
        <KeyvalImage
          src={data?.destination_type?.image_url}
          width={40}
          height={40}
          style={IMAGE_STYLE}
        />
        <TextWrapper>
          <KeyvalText size={14} weight={600}>
            {data?.destination_type?.display_name}
          </KeyvalText>
          <KeyvalText color={"#8b92a5"}>{data?.name}</KeyvalText>
        </TextWrapper>
      </NodeDataWrapper>
      <MonitorsListWrapper>{renderMonitors()}</MonitorsListWrapper>
      <Handle
        type="target"
        position={Position.Left}
        id="a"
        isConnectable={isConnectable}
        style={{ visibility: "hidden" }}
      />
    </DestinationNodeContainer>
  );
});
