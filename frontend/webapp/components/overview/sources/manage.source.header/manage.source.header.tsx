import React, { useRef, useState } from "react";
import styled from "styled-components";
import {
  KeyvalActionInput,
  KeyvalImage,
  KeyvalText,
  KeyvalTooltip,
} from "@/design.system";
import { Pen } from "@/assets/icons/overview";
import { useOnClickOutside } from "@/hooks";

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

const ActionInputWrapper = styled.div`
  width: 80%;
  height: 49px;
`;

const IMAGE_STYLE: React.CSSProperties = {
  backgroundColor: "#fff",
  padding: 4,
  marginRight: 16,
  marginLeft: 16,
};

export function ManageSourceHeader({ image_url, name }) {
  const [showEditInput, setShowEditInput] = useState(true);
  const [inputValue, setInputValue] = useState(name);
  const containerRef = useRef(null);
  const handleClickOutside = () => {
    !showEditInput && handleSave();
  };

  useOnClickOutside(containerRef, handleClickOutside);

  function handleSave() {
    setShowEditInput(true);
  }

  function handleInputChange(value) {
    setInputValue(value);
  }

  return (
    <ManageSourceHeaderWrapper ref={containerRef}>
      <KeyvalImage src={image_url} style={IMAGE_STYLE} />

      {showEditInput ? (
        <>
          <TextWrapper>
            <KeyvalText size={24} weight={700}>
              {name}
            </KeyvalText>
          </TextWrapper>

          <EditIconWrapper onClick={() => setShowEditInput(false)}>
            <Pen width={16} height={16} />
          </EditIconWrapper>
        </>
      ) : (
        <ActionInputWrapper>
          <KeyvalActionInput
            value={inputValue}
            onChange={(e) => handleInputChange(e)}
            onAction={handleSave}
          />
        </ActionInputWrapper>
      )}
    </ManageSourceHeaderWrapper>
  );
}
