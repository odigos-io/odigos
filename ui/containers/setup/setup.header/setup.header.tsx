import React from "react";
import {
  HeaderButtonWrapper,
  HeaderTitleWrapper,
  SetupHeaderWrapper,
} from "./setup.header.menu";
import { FloatBox, KeyvalText } from "design.system";
import Icon from "assets/icons/charge-rect.svg";
export function SetupHeader() {
  return (
    <SetupHeaderWrapper>
      <HeaderTitleWrapper>
        <Icon />
        <KeyvalText style={{ marginLeft: 24 }}>
          {"Select applications to connect"}
        </KeyvalText>
      </HeaderTitleWrapper>
      <HeaderButtonWrapper>
        <KeyvalText weight={400}>{"0 selected"}</KeyvalText>
      </HeaderButtonWrapper>
    </SetupHeaderWrapper>
  );
}
