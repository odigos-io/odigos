import React from "react";
import {
  HeaderButtonWrapper,
  HeaderTitleWrapper,
  SetupHeaderWrapper,
} from "./setup.header.styled";
import { KeyvalButton, KeyvalText } from "@/design.system";
import Charge from "assets/icons/charge-rect.svg";
import RightArrow from "assets/icons/arrow-right.svg";

export function SetupHeader({ onNextClick }: any) {
  return (
    <SetupHeaderWrapper>
      <HeaderTitleWrapper>
        <Charge />
        <KeyvalText>{"Select applications to connect"}</KeyvalText>
      </HeaderTitleWrapper>
      <HeaderButtonWrapper>
        <KeyvalText weight={400}>{"0 selected"}</KeyvalText>
        <KeyvalButton onClick={onNextClick} style={{ gap: 10 }}>
          <KeyvalText size={20} weight={600} color="#0A1824">
            {"Next"}
          </KeyvalText>
          <RightArrow />
        </KeyvalButton>
      </HeaderButtonWrapper>
    </SetupHeaderWrapper>
  );
}
