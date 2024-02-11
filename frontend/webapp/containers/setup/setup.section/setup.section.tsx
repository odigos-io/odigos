'use client';
import React, { useEffect, useRef, useState } from 'react';
import { STEPS, Step } from './utils';
import { WhiteArrow } from '@/assets/icons/app';
import { useSearchParams } from 'next/navigation';
import { KeyvalText, Steps } from '@/design.system';
import { SetupHeader } from '../setup.header/setup.header';
import { SourcesSection } from '../sources/sources.section';
import { CONFIG, NOTIFICATION, SETUP } from '@/utils/constants';
import { ConnectionSection } from '../connection/connection.section';
import { useSectionData, useNotification, useSources } from '@/hooks';
import { DestinationSection } from '../destination/destination.section';

import {
  SetupContentWrapper,
  SetupSectionContainer,
  StepListWrapper,
  BackButtonWrapper,
} from './setup.section.styled';
const STATE = 'state';

const sectionComponents = {
  [SETUP.STEPS.ID.CHOOSE_SOURCE]: SourcesSection,
  [SETUP.STEPS.ID.CHOOSE_DESTINATION]: DestinationSection,
  [SETUP.STEPS.ID.CREATE_CONNECTION]: ConnectionSection,
};

export function SetupSection() {
  const [currentStep, setCurrentStep] = useState<Step>(STEPS[0]);

  const { upsertSources } = useSources();
  const { show, Notification } = useNotification();
  const { sectionData, setSectionData, totalSelected } = useSectionData({});

  const searchParams = useSearchParams();
  const search = searchParams.get(STATE);
  const previousSourceState = useRef<any>(null);

  useEffect(() => {
    getStepFromSearch();
  }, [searchParams]);

  function getStepFromSearch() {
    if (search === CONFIG.APPS_SELECTED) {
      handleChangeStep(1);
    }
  }

  function renderCurrentSection() {
    const Component = sectionComponents[currentStep?.id];
    return Component ? (
      <Component
        sectionData={sectionData}
        // setSectionData={setSectionData}
        onSelectItem={onNextClick}
      />
    ) : null;
  }

  function handleChangeStep(direction: number) {
    const currentStepIndex = STEPS.findIndex(
      ({ id }) => id === currentStep?.id
    );
    const nextStep = STEPS[currentStepIndex + direction];
    const prevStep = STEPS[currentStepIndex];

    if (nextStep) {
      nextStep.status = SETUP.STEPS.STATUS.ACTIVE;
    }

    if (prevStep) {
      prevStep.status =
        direction === 1 ? SETUP.STEPS.STATUS.DONE : SETUP.STEPS.STATUS.DISABLED;
    }

    if (currentStep?.id === SETUP.STEPS.ID.CHOOSE_SOURCE) {
      previousSourceState.current = sectionData;

      upsertSources({
        sectionData,
        onSuccess: () => {
          setCurrentStep(nextStep);
          setSectionData({});
        },
        onError: ({ response }) => {
          const message = response?.data?.message || SETUP.ERROR;
          show({
            type: NOTIFICATION.ERROR,
            message,
          });
        },
      });

      return;
    }
    setCurrentStep(nextStep);
  }

  function onNextClick() {
    handleChangeStep(1);
  }

  function onBackClick() {
    handleChangeStep(-1);
    setSectionData(previousSourceState.current || {});
  }

  return (
    <>
      <StepListWrapper>
        <Steps data={STEPS} />
      </StepListWrapper>
      <SetupSectionContainer>
        {currentStep.index !== 1 && (
          <BackButtonWrapper onClick={onBackClick}>
            <WhiteArrow />
            <KeyvalText size={14} weight={600}>
              {SETUP.BACK}
            </KeyvalText>
          </BackButtonWrapper>
        )}
        <SetupHeader
          currentStep={currentStep}
          onNextClick={onNextClick}
          totalSelected={totalSelected}
          sectionData={sectionData}
        />
        <SetupContentWrapper>{renderCurrentSection()}</SetupContentWrapper>
      </SetupSectionContainer>
      <Notification />
    </>
  );
}
