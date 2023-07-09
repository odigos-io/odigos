import React from "react";
import { SelectedCounterWrapper } from "./selected.counter.styled";
import Checked from "@/assets/icons/check.svg";
import { KeyvalText } from "../text/text";

interface SelectedCounterProps {
  total: number;
  selected: number;
}

export function KeyvalSelectedCounter({
  total,
  selected,
}: SelectedCounterProps) {
  return (
    <SelectedCounterWrapper>
      {selected !== 0 && <Checked />}
      <KeyvalText size={13} weight={500}>{`${selected} / ${total}`}</KeyvalText>
    </SelectedCounterWrapper>
  );
}
