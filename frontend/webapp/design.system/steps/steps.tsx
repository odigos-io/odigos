import React from "react";
import StepItem from "./step.item";
import { StepsContainer } from "./steps.styled";

type StepListProps<T> = {
  data?: Array<T> | null;
};

export default function Steps<T>({ data }: StepListProps<T>) {
  function renderSteps() {
    return data?.map(({ title, status }: any, index) => (
      <StepItem
        key={`${index}_${title}`}
        title={title}
        status={status}
        index={index + 1}
        isLast={index + 1 === data.length}
      />
    ));
  }

  return <StepsContainer>{renderSteps()}</StepsContainer>;
}
