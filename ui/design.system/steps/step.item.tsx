import { FloatBox } from "design.system/float.box/float.box";
import { KeyvalText } from "design.system/text/text";
import React from "react";
import {
  StepItemTextWrapper,
  StepItemBorder,
  StepItemWrapper,
} from "./steps.styled";

type StepItemProps = {
  title: string;
  index: number;
  status: string;
};

enum Status {
  Done = "done",
  Active = "active",
  Disabled = "disabled",
}

export default function StepItem({ title, index, status }: StepItemProps) {
  console.log({ status });
  return (
    <StepItemWrapper>
      <FloatBox label={index} />
      <StepItemTextWrapper>
        <KeyvalText>{title}</KeyvalText>
      </StepItemTextWrapper>
      <StepItemBorder />
    </StepItemWrapper>
  );
}
