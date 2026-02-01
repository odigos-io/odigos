'use client';

import React, { useMemo, type PropsWithChildren } from 'react';
import { usePathname } from 'next/navigation';
import { ROUTES } from '@/utils';
import styled from 'styled-components';
import { ToastList } from '@odigos/ui-kit/containers';
import { OnboardingStepperWrapper } from '@/components';
import { OdigosProvider } from '@odigos/ui-kit/contexts';
import { DISPLAY_TITLES } from '@odigos/ui-kit/constants';
import { useConfig, useSSE, useTokenTracker } from '@/hooks';
import { useDataStreamStore, useSetupStore } from '@odigos/ui-kit/store';
import { ErrorBoundary, FlexColumn, Stepper, StepperProps } from '@odigos/ui-kit/components';

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

  const { selectedStreamName } = useDataStreamStore();
  const { configuredSources, configuredDestinations, configuredDestinationsUpdateOnly } = useSetupStore();

  const pathname = usePathname();
  const { config } = useConfig();

  const sourceCount = useMemo(() => Object.values(configuredSources).reduce((total, sourceList) => total + sourceList.filter((s) => s.selected).length, 0), [configuredSources]);
  const destCount = useMemo(() => configuredDestinations.length + configuredDestinationsUpdateOnly.length, [configuredDestinations, configuredDestinationsUpdateOnly]);

  const stepsData: StepperProps['data'] = useMemo(
    () => [
      { stepNumber: 1, title: DISPLAY_TITLES.INSTALLATION },
      { stepNumber: 2, title: DISPLAY_TITLES.DATA_STREAM, subtitle: selectedStreamName },
      { stepNumber: 3, title: DISPLAY_TITLES.SOURCES, subtitle: `${sourceCount} ${DISPLAY_TITLES.SOURCES}` },
      { stepNumber: 4, title: DISPLAY_TITLES.DESTINATIONS, subtitle: `${destCount} ${DISPLAY_TITLES.DESTINATIONS}` },
      { stepNumber: 5, title: DISPLAY_TITLES.SUMMARY },
    ],
    [selectedStreamName, sourceCount, destCount],
  );

  return (
    <ErrorBoundary>
      <OdigosProvider platformType={config?.platformType} tier={config?.tier} version={config?.odigosVersion || ''}>
        <PageContent>
          <OnboardingStepperWrapper>
            <Stepper currentStep={steps[pathname] || 1} data={stepsData} />
          </OnboardingStepperWrapper>

          {children}

          <ToastList />
        </PageContent>
      </OdigosProvider>
    </ErrorBoundary>
  );
}

export default SetupLayout;
