import React from "react";
import { SelectedCounter } from "@keyval-org/design-system";

interface SelectedCounterProps {
  total: number;
  selected: number;
}

export function KeyvalSelectedCounter(props: SelectedCounterProps) {
  return <SelectedCounter {...props} />;
}
