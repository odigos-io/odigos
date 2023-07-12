import React from "react";
import {
  HeaderButtonWrapper,
  HeaderTitleWrapper,
  SetupHeaderWrapper,
} from "./setup.header.styled";
import { KeyvalButton, KeyvalText } from "@/design.system";
import Charge from "assets/icons/charge-rect.svg";
import Connect from "assets/icons/connect.svg";
import RightArrow from "assets/icons/arrow-right.svg";
import { SETUP } from "@/utils/constants";

export function SetupHeader({ currentStep, onNextClick, totalSelected }: any) {
  function renderCurrentIcon() {
    switch (currentStep?.id) {
      case "choose-source":
        return (
          <>
            <Charge />
            <KeyvalText>{SETUP.HEADER.CHOOSE_SOURCE_TITLE}</KeyvalText>
          </>
        );
      case "choose-destination":
        return (
          <>
            <Connect />
            <KeyvalText>{SETUP.HEADER.CHOOSE_DESTINATION_TITLE}</KeyvalText>
          </>
        );
      default:
        return null;
    }
  }

  return (
    <SetupHeaderWrapper>
      <HeaderTitleWrapper>{renderCurrentIcon()}</HeaderTitleWrapper>
      <HeaderButtonWrapper>
        {currentStep?.id === "choose-source" && (
          <KeyvalText
            weight={400}
          >{`${totalSelected} ${SETUP.SELECTED}`}</KeyvalText>
        )}
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
