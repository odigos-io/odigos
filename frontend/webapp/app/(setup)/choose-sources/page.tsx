'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import Theme from '@odigos/ui-theme';
import { useAppStore } from '@/store';
import { useSourceFormData, useSSE } from '@/hooks';
import { OnboardingStepperWrapper } from '@/styles';
import { ChooseSourcesBody } from '@/containers/main';
import { ArrowIcon, OdigosLogoText } from '@odigos/ui-icons';
import { FlexRow, Header, NavigationButtons, Stepper, Text } from '@odigos/ui-components';

export default function Page() {
  // call important hooks that should run on page-mount
  useSSE();

  const router = useRouter();
  const appState = useAppStore();
  const menuState = useSourceFormData();

  const onNext = () => {
    const { recordedInitialSources, getApiSourcesPayload, getApiFutureAppsPayload } = menuState;
    const { setAvailableSources, setConfiguredSources, setConfiguredFutureApps } = appState;

    setAvailableSources(recordedInitialSources);
    setConfiguredSources(getApiSourcesPayload());
    setConfiguredFutureApps(getApiFutureAppsPayload());

    router.push(ROUTES.CHOOSE_DESTINATION);
  };

  return (
    <>
      <Header
        left={[<OdigosLogoText size={100} />]}
        center={[<Text family='secondary'>START WITH ODIGOS</Text>]}
        right={[
          <FlexRow>
            <Theme.ToggleDarkMode />
            <NavigationButtons
              buttons={[
                {
                  label: 'NEXT',
                  icon: ArrowIcon,
                  onClick: () => onNext(),
                  variant: 'primary',
                },
              ]}
            />
          </FlexRow>,
        ]}
      />

      <OnboardingStepperWrapper>
        <Stepper
          currentStep={2}
          data={[
            { stepNumber: 1, title: 'INSTALLATION' },
            { stepNumber: 2, title: 'SOURCES' },
            { stepNumber: 3, title: 'DESTINATIONS' },
          ]}
        />
      </OnboardingStepperWrapper>

      <ChooseSourcesBody componentType='FAST' {...menuState} />
    </>
  );
}
