'use client';
import React from 'react';
import { Steps } from '@/design.system';

const STEP_STATUS = {
  ACTIVE: 'active',
  DISABLED: 'disabled',
  DONE: 'done',
};

const SETUP_STEPS = [
  {
    title: 'Choose Source',
    index: 0,
    status: 'disabled',
    id: 'choose-source',
  },
  {
    title: 'Choose Destination',
    index: 1,
    status: 'disabled',
    id: 'choose-destination',
  },
  {
    title: 'Create Connection',
    index: 2,
    status: 'disabled',
    id: 'create-connection',
  },
];

type Step = {
  index: number;
  title: string;
  status: string;
  id: string;
  subtitle?: string;
};

interface SetupStepsProps {
  currentStepIndex: number;
}

export function StepsList({ currentStepIndex }: SetupStepsProps) {
  function buildSteps() {
    return SETUP_STEPS.map((step: Step) => {
      return {
        ...step,
        status:
          step.index === currentStepIndex
            ? STEP_STATUS.ACTIVE
            : step.index < currentStepIndex
            ? STEP_STATUS.DONE
            : STEP_STATUS.DISABLED,
      };
    });
  }

  return <Steps data={buildSteps()} />;
}
