import React, { useState } from "react";
import styled from "styled-components";
import { KeyvalImage, KeyvalText, KeyvalTooltip } from "@/design.system";
import { Pen } from "@/assets/icons/overview";

const ManageSourceHeaderWrapper = styled.div`
  display: flex;
  width: 100%;
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

const TextWrapper = styled.div`
  margin-left: 12px;
  margin-right: 12px;
`;

const EditIconWrapper = styled.div`
  width: 20px;
  height: 40px;
  display: flex;
  align-items: center;

  :hover {
    cursor: pointer;
    fill: ${({ theme }) => theme.colors.secondary};
  }
`;
const IMAGE_STYLE: React.CSSProperties = {
  backgroundColor: "#fff",
  padding: 4,
  marginRight: 16,
  marginLeft: 16,
};

export function ManageSourceHeader({ image_url, display_name }) {
  const [showEditInput, setShowEditInput] = useState(true);
  return (
    <ManageSourceHeaderWrapper>
      <KeyvalImage src={image_url} style={IMAGE_STYLE} />
      <TextWrapper>
        <KeyvalText size={24} weight={700}>
          {display_name}
        </KeyvalText>
      </TextWrapper>
      {showEditInput ? (
        <EditIconWrapper onClick={() => setShowEditInput(false)}>
          <Pen width={16} height={16} />
        </EditIconWrapper>
      ) : null}
    </ManageSourceHeaderWrapper>
  );
}
