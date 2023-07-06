import { FloatBox } from "@/design.system/float.box/float.box";
import { KeyvalText } from "@/design.system/text/text";
import React from "react";
import {
  StepItemTextWrapper,
  StepItemBorder,
  StepItemWrapper,
  FloatingBoxTextWrapper,
} from "./steps.styled";
import Done from "assets/icons/checked.svg";

type StepItemProps = {
  title: string;
  index: number;
  status: string;
  isLast: boolean;
};

enum Status {
  Done = "done",
  Active = "active",
  Disabled = "disabled",
}

export default function StepItem({
  title,
  index,
  status,
  isLast,
}: StepItemProps) {
  return (
    <StepItemWrapper>
      <FloatBox>
        {status === Status.Done ? (
          <Done />
        ) : (
          <FloatingBoxTextWrapper disabled={status !== Status.Active}>
            <KeyvalText weight={700}>{index}</KeyvalText>
          </FloatingBoxTextWrapper>
        )}
      </FloatBox>
      <StepItemTextWrapper disabled={status !== Status.Active}>
        <KeyvalText weight={600}>{title}</KeyvalText>
      </StepItemTextWrapper>
      {!isLast && <StepItemBorder />}
    </StepItemWrapper>
  );
}
