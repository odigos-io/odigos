import React from "react";
import styled from "styled-components";
import { KeyvalImage } from "@/design.system";

const ManageSourceHeaderWrapper = styled.div`
  display: flex;
  width: 100%;
  min-width: 686px;
  height: 104px;
  align-items: center;
  border-radius: 25px;
  margin: 24px 0;
  background: radial-gradient(
      78.09% 72.18% at 100% -0%,
      rgba(150, 242, 255, 0.4) 0%,
      rgba(150, 242, 255, 0) 61.91%
    ),
    linear-gradient(180deg, #2e4c55 0%, #303355 100%);
`;

const IMAGE_STYLE: React.CSSProperties = {
  backgroundColor: "#fff",
  padding: 4,
  marginRight: 16,
  marginLeft: 16,
};

export function ManageSourceHeader({ image_url }) {
  return (
    <ManageSourceHeaderWrapper>
      <KeyvalImage src={image_url} style={IMAGE_STYLE} />
    </ManageSourceHeaderWrapper>
  );
}
