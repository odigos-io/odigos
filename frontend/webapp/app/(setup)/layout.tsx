'use client';

import React, { type PropsWithChildren } from 'react';
import { usePathname } from 'next/navigation';
import { ROUTES } from '@/utils';
import styled from 'styled-components';
import { ToastList } from '@odigos/ui-kit/containers';
import { OnboardingStepperWrapper } from '@/components';
import { DISPLAY_TITLES } from '@odigos/ui-kit/constants';
import { useDataStreamsCRUD, useSSE, useTokenTracker } from '@/hooks';
import { ErrorBoundary, FlexColumn, Stepper } from '@odigos/ui-kit/components';

const PageContent = styled(FlexColumn)`
  width: 100%;
  height: 100vh;
  background-color: ${({ theme }) => theme.colors?.primary};
  align-items: center;
`;

const steps = {
  [ROUTES.CHOOSE_STREAM]: 2,
  [ROUTES.CHOOSE_SOURCES]: 3,
  [ROUTES.CHOOSE_DESTINATION]: 4,
  [ROUTES.SETUP_SUMMARY]: 5,
};

function SetupLayout({ children }: PropsWithChildren) {
  // call important hooks that should run on page-mount
  useSSE();
  useTokenTracker();
  useDataStreamsCRUD();

  const pathname = usePathname();

  return (
    <ErrorBoundary>
      <PageContent>
        <OnboardingStepperWrapper>
          <Stepper
            currentStep={steps[pathname] || 1}
            data={[
              { stepNumber: 1, title: DISPLAY_TITLES.INSTALLATION },
              { stepNumber: 2, title: DISPLAY_TITLES.DATA_STREAM },
              { stepNumber: 3, title: DISPLAY_TITLES.SOURCES },
              { stepNumber: 4, title: DISPLAY_TITLES.DESTINATIONS },
              { stepNumber: 5, title: DISPLAY_TITLES.SUMMARY },
            ]}
          />
        </OnboardingStepperWrapper>

        {children}

        <ToastList />
      </PageContent>
    </ErrorBoundary>
  );
}

export default SetupLayout;
