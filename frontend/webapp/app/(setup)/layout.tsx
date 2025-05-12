'use client';

import React, { type PropsWithChildren } from 'react';
import { usePathname } from 'next/navigation';
import { ROUTES } from '@/utils';
import styled from 'styled-components';
import { ToastList } from '@odigos/ui-kit/containers';
import { OnboardingStepperWrapper } from '@/components';
import { useDataStreamsCRUD, useSSE, useTokenTracker } from '@/hooks';
import { ErrorBoundary, FlexColumn, Stepper } from '@odigos/ui-kit/components';

const PageContent = styled(FlexColumn)`
  width: 100%;
  height: 100vh;
  background-color: ${({ theme }) => theme.colors?.primary};
  align-items: center;
`;

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
            currentStep={pathname === ROUTES.CHOOSE_STREAM ? 2 : pathname === ROUTES.CHOOSE_SOURCES ? 3 : pathname === ROUTES.CHOOSE_DESTINATION ? 4 : pathname === ROUTES.SETUP_SUMMARY ? 5 : 1}
            data={[
              { stepNumber: 1, title: 'INSTALLATION' },
              { stepNumber: 2, title: 'DATA STREAM' },
              { stepNumber: 3, title: 'SOURCES' },
              { stepNumber: 4, title: 'DESTINATIONS' },
              { stepNumber: 5, title: 'SUMMARY' },
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
