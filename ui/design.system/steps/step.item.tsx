import { FloatBox } from "design.system/float.box/float.box";
import { KeyvalText } from "design.system/text/text";
import React from "react";
import styled from "styled-components";

type StepItemProps = {
  title: string;
  index: number;
};

const StepItemWraaper = styled.div`
  display: flex;
  align-items: center;
`;

const StepItemTextWraaper = styled.div`
  margin: 0 8px;
`;

const StepItemBorder = styled.div`
  width: 54px;
  height: 1px;
  background-color: #8b92a5;
  margin-top: 2px;
`;

export default function StepItem({ title, index }: StepItemProps) {
  return (
    <StepItemWraaper>
      <FloatBox label="1" />
      <StepItemTextWraaper>
        <KeyvalText>{"Choose Source"}</KeyvalText>
      </StepItemTextWraaper>
      <StepItemBorder />
    </StepItemWraaper>
  );
}
