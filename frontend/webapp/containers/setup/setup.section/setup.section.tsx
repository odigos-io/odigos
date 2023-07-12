import React, { useMemo, useState } from "react";
import {
  SetupContentWrapper,
  SetupSectionContainer,
  StepListWrapper,
} from "./setup.section.styled";
import { SetupHeader } from "../setup.header/setup.header";
import { SETUP } from "@/utils/constants";
import Steps from "@/design.system/steps/steps";
import { SourcesSection } from "../sources/sources.section";
import { DestinationSection } from "../destination/destination.section";

const STEPS = [
  {
    index: 1,
    id: "choose-source",
    title: SETUP.STEPS.CHOOSE_SOURCE,
    status: SETUP.STEPS.STATUS.ACTIVE,
  },
  {
    index: 2,
    id: "choose-destination",
    title: SETUP.STEPS.CHOOSE_DESTINATION,
    status: SETUP.STEPS.STATUS.DISABLED,
  },
  {
    index: 3,
    id: "create-connection",
    title: SETUP.STEPS.CREATE_CONNECTION,
    status: SETUP.STEPS.STATUS.DISABLED,
  },
];

export function SetupSection() {
  const [sectionData, setSectionData] = useState<any>({});
  const [steps, setSteps] = useState(STEPS);
  const [currentStep, setCurrentStep] = useState(STEPS[0]);

  const totalSelected = useMemo(() => {
    let total = 0;
    for (const key in sectionData) {
      const apps = sectionData[key]?.objects;
      const counter = apps?.filter((item: any) => item.selected)?.length;
      total += counter;
    }
    return total;
  }, [JSON.stringify(sectionData)]);

  function renderCurrentSection() {
    switch (currentStep?.id) {
      case "choose-source":
        return (
          <SourcesSection
            sectionData={sectionData}
            setSectionData={setSectionData}
          />
        );
      case "choose-destination":
        return <DestinationSection />;
      default:
        return null;
    }
  }

  function handleNamespacesUpdate() {
    // setNamespaces(sectionData);
    setCurrentStep(STEPS[1]);
    setSectionData({});
  }

  function onNextClick() {
    switch (currentStep?.id) {
      case "choose-source":
        handleNamespacesUpdate();
      default:
        return null;
    }
  }

  return (
    <>
      <StepListWrapper>
        <Steps data={steps} />
      </StepListWrapper>
      <SetupSectionContainer>
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
