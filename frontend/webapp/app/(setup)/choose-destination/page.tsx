'use client';
import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import Theme, { styled } from '@odigos/ui-theme';
import { OnboardingStepperWrapper } from '@/styles';
import { ConfiguredDestinationsList } from '@/containers';
import { ENTITY_TYPES, NOTIFICATION_TYPE } from '@odigos/ui-utils';
import { ArrowIcon, OdigosLogoText, PlusIcon } from '@odigos/ui-icons';
import { DestinationModal, useModalStore, useSetupStore } from '@odigos/ui-containers';
import { useDestinationCategories, useDestinationCRUD, usePotentialDestinations, useSourceCRUD, useSSE, useTestConnection } from '@/hooks';
import { Button, CenterThis, FadeLoader, Header, NavigationButtons, NotificationNote, SectionTitle, Stepper, Text } from '@odigos/ui-components';

const ContentWrapper = styled.div`
  width: 640px;
  padding-top: 64px;
`;

const NotificationNoteWrapper = styled.div`
  margin-top: 24px;
`;

const AddDestinationButtonWrapper = styled.div`
  width: 100%;
  margin-top: 24px;
`;

const StyledAddDestinationButton = styled(Button)`
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
`;

export default function Page() {
  // call important hooks that should run on page-mount
  useSSE();

  const theme = Theme.useTheme();
  const router = useRouter();
  const { setCurrentModal } = useModalStore();

  const onOpen = () => setCurrentModal(ENTITY_TYPES.DESTINATION);

  const { persistSources } = useSourceCRUD();
  const { categories } = useDestinationCategories();
  const { createDestination } = useDestinationCRUD();
  const { potentialDestinations } = usePotentialDestinations();
  const { testConnection, loading: testLoading, data: testResult } = useTestConnection();
  const { configuredSources, configuredFutureApps, configuredDestinations, resetState } = useSetupStore();

  // we need this state, because "loading" from CRUD hooks is a bit delayed, and allows the user to double-click, as well as see elements render in the UI when they should not be rendered.
  const [isLoading, setIsLoading] = useState(false);

  const clickBack = () => {
    router.push(ROUTES.CHOOSE_SOURCES);
  };

  const clickDone = async () => {
    setIsLoading(true);

    // configuredSources & configuredFutureApps are set in store from the previous step in onboarding flow
    await persistSources(configuredSources, configuredFutureApps);
    await Promise.all(configuredDestinations.map(async ({ form }) => await createDestination(form)));

    resetState();
    router.push(ROUTES.OVERVIEW);
  };

  const isSourcesListEmpty = () => !Object.values(configuredSources).some((sources) => !!sources.length);

  return (
    <>
      <Header
        left={[<OdigosLogoText key='logo' size={100} />]}
        center={[
          <Text key='msg' family='secondary'>
            START WITH ODIGOS
          </Text>,
        ]}
        right={[
          <Theme.ToggleDarkMode key='toggle-theme' />,
          <NavigationButtons
            key='nav-buttons'
            buttons={[
              {
                label: 'BACK',
                icon: ArrowIcon,
                variant: 'secondary',
                onClick: clickBack,
                disabled: isLoading,
              },
              {
                label: 'DONE',
                variant: 'primary',
                onClick: clickDone,
                disabled: isLoading,
              },
            ]}
          />,
        ]}
      />

      <OnboardingStepperWrapper>
        <Stepper
          currentStep={3}
          data={[
            { stepNumber: 1, title: 'INSTALLATION' },
            { stepNumber: 2, title: 'SOURCES' },
            { stepNumber: 3, title: 'DESTINATIONS' },
          ]}
        />
      </OnboardingStepperWrapper>

      <ContentWrapper>
        <SectionTitle title='Configure destinations' description='Select destinations where telemetry data will be sent and configure their settings.' />

        {!isLoading && isSourcesListEmpty() && (
          <NotificationNoteWrapper>
            <NotificationNote
              type={NOTIFICATION_TYPE.WARNING}
              message='No sources selected. Please go back to select sources.'
              action={{
                label: 'Select sources',
                onClick: () => router.push(ROUTES.CHOOSE_SOURCES),
              }}
            />
          </NotificationNoteWrapper>
        )}

        <AddDestinationButtonWrapper>
          <StyledAddDestinationButton variant='secondary' disabled={isLoading} onClick={onOpen}>
            <PlusIcon />
            <Text color={theme.colors.secondary} size={14} decoration='underline' family='secondary'>
              ADD DESTINATION
            </Text>
          </StyledAddDestinationButton>

          <DestinationModal
            isOnboarding={true}
            categories={categories}
            potentialDestinations={potentialDestinations}
            createDestination={createDestination}
            testConnection={testConnection}
            testLoading={testLoading}
            testResult={testResult}
          />
        </AddDestinationButtonWrapper>

        {isLoading ? (
          <CenterThis>
            <FadeLoader scale={2} cssOverride={{ marginTop: '3rem' }} />
          </CenterThis>
        ) : (
          <ConfiguredDestinationsList data={configuredDestinations} />
        )}
      </ContentWrapper>
    </>
  );
}
