import FloatBox from "design.system/float.box/float.box";
import React from "react";
import styled from "styled-components";

type StepItemPros = {
  title: string;
  index: number;
};

const StepItemWraaper = styled.div`
  display: flex;
`;

export default function StepItem({ title, index }: StepItemPros) {
  return (
    <StepItemWraaper>
      <FloatBox label="tets" />
    </StepItemWraaper>
  );
}
