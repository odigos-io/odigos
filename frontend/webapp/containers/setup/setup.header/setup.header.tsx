import React from "react";
import {
  HeaderButtonWrapper,
  HeaderTitleWrapper,
  SetupHeaderWrapper,
} from "./setup.header.styled";
import { KeyvalButton, KeyvalText } from "@/design.system";
import Charge from "assets/icons/charge-rect.svg";
import RightArrow from "assets/icons/arrow-right.svg";
import { SETUP } from "@/utils/constants";

export function SetupHeader({ onNextClick, totalSelected }: any) {
  return (
    <SetupHeaderWrapper>
      <HeaderTitleWrapper>
        <Charge />
        <KeyvalText>{SETUP.HEADER.CHOOSE_SOURCE_TITLE}</KeyvalText>
      </HeaderTitleWrapper>
      <HeaderButtonWrapper>
        <KeyvalText
          weight={400}
        >{`${totalSelected} ${SETUP.SELECTED}`}</KeyvalText>
        <KeyvalButton
          disabled={totalSelected === 0}
          onClick={onNextClick}
          style={{ gap: 10 }}
        >
          <KeyvalText size={20} weight={600} color="#0A1824">
            {SETUP.NEXT}
          </KeyvalText>
          <RightArrow />
        </KeyvalButton>
      </HeaderButtonWrapper>
    </SetupHeaderWrapper>
  );
}
