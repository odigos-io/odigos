import React from "react";
import { SelectedCounter } from "@odigos-io/design-system";

interface SelectedCounterProps {
  total: number;
  selected: number;
}

export function KeyvalSelectedCounter(props: SelectedCounterProps) {
  return <SelectedCounter {...props} />;
}
