import React, { useState } from "react";
import {
  SetupContentWrapper,
  SetupSectionContainer,
  StepListWrapper,
  BackButtonWrapper,
} from "./setup.section.styled";
import { SetupHeader } from "../setup.header/setup.header";
import { DestinationSection } from "../destination/destination.section";
import { ConnectionSection } from "../connection/connection.section";
import { SourcesSection } from "../sources/sources.section";
import RightArrow from "assets/icons/white-arrow-right.svg";
import Steps from "@/design.system/steps/steps";
import { KeyvalText } from "@/design.system";
import { SETUP } from "@/utils/constants";
import { useSectionData } from "@/hooks";
import { STEPS, Step } from "./utils";

const sectionComponents = {
  [SETUP.STEPS.ID.CHOOSE_SOURCE]: SourcesSection,
  [SETUP.STEPS.ID.CHOOSE_DESTINATION]: DestinationSection,
  [SETUP.STEPS.ID.CREATE_CONNECTION]: ConnectionSection,
};

export function SetupSection() {
  const [steps, setSteps] = useState<Step[]>(STEPS);
  const [currentStep, setCurrentStep] = useState<Step>(STEPS[0]);
  const { sectionData, setSectionData, totalSelected } = useSectionData({});

  function renderCurrentSection() {
    const Component = sectionComponents[currentStep?.id];
    return Component ? (
      <Component sectionData={sectionData} setSectionData={setSectionData} />
    ) : null;
  }

  function handleChangeStep(direction: number) {
    const currentStepIndex = steps.findIndex(
      ({ id }) => id === currentStep?.id
    );
    const nextStep = steps[currentStepIndex + direction];
    const prevStep = steps[currentStepIndex];

    if (nextStep) {
      nextStep.status = SETUP.STEPS.STATUS.ACTIVE;
    }

    if (prevStep && direction === 1) {
      prevStep.status = SETUP.STEPS.STATUS.DONE;
    } else {
      prevStep.status = SETUP.STEPS.STATUS.DISABLED;
    }
    setCurrentStep(nextStep);
    setSteps([...steps]);
    // setSectionData({});
  }

  function onNextClick() {
    handleChangeStep(1);
  }

  function onBackClick() {
    handleChangeStep(-1);
  }

  return (
    <>
      <StepListWrapper>
        <Steps data={steps} />
      </StepListWrapper>
      <SetupSectionContainer>
        {currentStep.index !== 1 && (
          <BackButtonWrapper onClick={onBackClick}>
            <RightArrow />
            <KeyvalText size={14} weight={600}>
              {SETUP.BACK}
            </KeyvalText>
          </BackButtonWrapper>
        )}
        <SetupHeader
          currentStep={currentStep}
          onNextClick={onNextClick}
          totalSelected={totalSelected}
        />
        <SetupContentWrapper>{renderCurrentSection()}</SetupContentWrapper>
      </SetupSectionContainer>
    </>
  );
}
