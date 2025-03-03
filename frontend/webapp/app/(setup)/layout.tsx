'use client';

import React, { type PropsWithChildren } from 'react';
import { usePathname } from 'next/navigation';
import { ROUTES } from '@/utils';
import styled from 'styled-components';
import { useSSE, useTokenTracker } from '@/hooks';
import { ToastList } from '@odigos/ui-containers';
import { FlexColumn, Stepper } from '@odigos/ui-components';
import { ErrorBoundary, OnboardingStepperWrapper } from '@/components';

const PageContent = styled(FlexColumn)`
  width: 100%;
  height: 100vh;
  background-color: ${({ theme }) => theme.colors.primary};
  align-items: center;
`;

function SetupLayout({ children }: PropsWithChildren) {
  // call important hooks that should run on page-mount
  useSSE();
  useTokenTracker();

  const pathname = usePathname();

  return (
    <ErrorBoundary>
      <PageContent>
        <OnboardingStepperWrapper>
          <Stepper
            currentStep={pathname === ROUTES.CHOOSE_SOURCES ? 2 : pathname === ROUTES.CHOOSE_DESTINATION ? 3 : 1}
            data={[
              { stepNumber: 1, title: 'INSTALLATION' },
              { stepNumber: 2, title: 'SOURCES' },
              { stepNumber: 3, title: 'DESTINATIONS' },
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
