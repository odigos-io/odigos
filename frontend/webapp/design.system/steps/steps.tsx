import React from "react";
import { Steps } from "@keyval-org/design-system";

type StepItemProps = {
  title: string;
  index: number;
  status: string;
  isLast?: boolean;
};

type StepListProps = {
  data?: StepItemProps[] | null;
};

export function KeyvalSteps(props: StepListProps) {
  return <Steps {...props} />;
}
