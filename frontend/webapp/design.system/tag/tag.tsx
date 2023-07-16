import { KeyvalText } from "@/design.system/text/text";
import React from "react";
import styled from "styled-components";

interface TagProps {
  title: string;
  color?: string;
}

const TagWrapper = styled.div`
  display: flex;
  padding: 4px 8px;
  align-items: flex-start;
  gap: 10px;
  border-radius: 10px;
`;

export function KeyvalTag({ title = "", color = "#033869" }: TagProps) {
  return (
    <TagWrapper style={{ backgroundColor: color }}>
      <KeyvalText weight={500} size={13} color={"#CCD0D2"}>
        {title}
      </KeyvalText>
    </TagWrapper>
  );
}
