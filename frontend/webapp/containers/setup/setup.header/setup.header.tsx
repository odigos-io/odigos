import React, { ReactNode } from "react";
import Charge from "assets/icons/charge-rect.svg";
import Connect from "assets/icons/connect.svg";
import RightArrow from "assets/icons/arrow-right.svg";
import {
  HeaderButtonWrapper,
  HeaderTitleWrapper,
  SetupHeaderWrapper,
} from "./setup.header.styled";
import { KeyvalButton, KeyvalText } from "@/design.system";
import { SETUP } from "@/utils/constants";

type StepId = "CHOOSE_SOURCE" | "CHOOSE_DESTINATION";

type SetupStep = {
  id?: StepId;
};

type SetupHeaderProps = {
  currentStep: any;
  onNextClick: () => void;
  totalSelected: number;
};

const renderCurrentIcon = (currentStep: SetupStep | null): ReactNode => {
  const { STEPS, HEADER } = SETUP;
  const { id }: SetupStep = currentStep || {};
  switch (id) {
    case STEPS.ID.CHOOSE_SOURCE:
      return (
        <>
          <Charge />
          <KeyvalText>{HEADER.CHOOSE_SOURCE_TITLE}</KeyvalText>
        </>
      );
    case STEPS.ID.CHOOSE_DESTINATION:
      return (
        <>
          <Connect />
          <KeyvalText>{HEADER.CHOOSE_DESTINATION_TITLE}</KeyvalText>
        </>
      );
    default:
      return null;
  }
};

export function SetupHeader({
  currentStep,
  onNextClick,
  totalSelected,
}: SetupHeaderProps) {
  return (
    <SetupHeaderWrapper>
      <HeaderTitleWrapper>{renderCurrentIcon(currentStep)}</HeaderTitleWrapper>
      <HeaderButtonWrapper>
        {currentStep?.id === SETUP.STEPS.ID.CHOOSE_SOURCE && (
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
