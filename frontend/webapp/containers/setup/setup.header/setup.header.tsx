import React, { ReactNode } from "react";
import Charge from "assets/icons/charge-rect.svg";
import Connect from "assets/icons/connect.svg";
import RightArrow from "assets/icons/arrow-right.svg";
import {
  HeaderButtonWrapper,
  HeaderTitleWrapper,
  SetupHeaderWrapper,
  TotalSelectedWrapper,
} from "./setup.header.styled";
import { KeyvalButton, KeyvalText } from "@/design.system";
import { SETUP } from "@/utils/constants";
import { ConnectionsIcons } from "@/components/setup";
import theme from "@/styles/palette";

type StepId = "CHOOSE_SOURCE" | "CHOOSE_DESTINATION";

type SetupStep = {
  id?: StepId;
};

type SetupHeaderProps = {
  currentStep: any;
  onNextClick: () => void;
  totalSelected: number;
  sectionData: any;
};

const renderCurrentIcon = (
  currentStep: SetupStep | null,
  data: any | undefined
): ReactNode => {
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
    case STEPS.ID.CREATE_CONNECTION:
      return (
        <>
          <ConnectionsIcons icon={data?.image_url} />
          <KeyvalText
            size={20}
            weight={600}
          >{`Add ${data?.display_name} Destination`}</KeyvalText>
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
  sectionData,
}: SetupHeaderProps) {
  return (
    <SetupHeaderWrapper>
      <HeaderTitleWrapper>
        {renderCurrentIcon(currentStep, sectionData)}
      </HeaderTitleWrapper>
      <HeaderButtonWrapper>
        {currentStep?.id === SETUP.STEPS.ID.CHOOSE_SOURCE &&
          !isNaN(totalSelected) && (
            <TotalSelectedWrapper>
              <KeyvalText>{totalSelected}</KeyvalText>
              <KeyvalText>{SETUP.SELECTED}</KeyvalText>
            </TotalSelectedWrapper>
          )}
        {currentStep?.id !== SETUP.STEPS.ID.CREATE_CONNECTION && (
          <KeyvalButton
            disabled={totalSelected === 0}
            onClick={onNextClick}
            style={{ gap: 10, width: 120 }}
          >
            <KeyvalText size={20} weight={600} color={theme.text.dark_button}>
              {SETUP.NEXT}
            </KeyvalText>
            <RightArrow />
          </KeyvalButton>
        )}
      </HeaderButtonWrapper>
    </SetupHeaderWrapper>
  );
}
