import React, { useState, type FC, type RefObject } from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import { EntityTypes } from '@odigos/ui-kit/types';
import { useSetupStore } from '@odigos/ui-kit/store';
import { useDestinationCRUD, useSourceCRUD } from '@/hooks';
import { ArrowIcon, OdigosLogoText } from '@odigos/ui-kit/icons';
import { ToggleDarkMode, type SourceSelectionFormRef } from '@odigos/ui-kit/containers';
import { Header, NavigationButtons, NavigationButtonsProps, Text } from '@odigos/ui-kit/components';

interface SetupHeaderProps {
  entityType: EntityTypes;
  formRef?: RefObject<SourceSelectionFormRef | null>; // in sources
}

const SetupHeader: FC<SetupHeaderProps> = ({ formRef, entityType }) => {
  const router = useRouter();

  const { persistSources } = useSourceCRUD();
  const { createDestination } = useDestinationCRUD();
  const { configuredSources, configuredFutureApps, configuredDestinations, setAvailableSources, setConfiguredSources, setConfiguredFutureApps, resetState } = useSetupStore();

  const [isLoading, setIsLoading] = useState(false);

  const onNext = () => {
    if (formRef?.current) {
      const { initial, apps, futureApps } = formRef.current.getFormValues();

      setAvailableSources(initial);
      setConfiguredSources(apps);
      setConfiguredFutureApps(futureApps);

      router.push(ROUTES.CHOOSE_DESTINATION);
    }
  };

  const onBack = () => {
    router.push(ROUTES.CHOOSE_SOURCES);
  };

  const onDone = async () => {
    setIsLoading?.(true);

    // configuredSources & configuredFutureApps are set in store from the previous step in onboarding flow
    await persistSources(configuredSources, configuredFutureApps);
    await Promise.all(configuredDestinations.map(async ({ form }) => await createDestination(form)));

    resetState();
    router.push(ROUTES.OVERVIEW);
  };

  const navButtons: NavigationButtonsProps['buttons'] =
    entityType === EntityTypes.Source
      ? [
          {
            label: 'NEXT',
            icon: ArrowIcon,
            onClick: () => onNext(),
            variant: 'primary',
          },
        ]
      : entityType === EntityTypes.Destination
      ? [
          {
            label: 'BACK',
            icon: ArrowIcon,
            variant: 'secondary',
            onClick: onBack,
            disabled: isLoading,
          },
          {
            label: 'DONE',
            variant: 'primary',
            onClick: onDone,
            disabled: isLoading,
          },
        ]
      : [];

  return (
    <Header
      left={[<OdigosLogoText key='logo' size={100} />]}
      center={[
        <Text key='msg' family='secondary'>
          START WITH ODIGOS
        </Text>,
      ]}
      right={[<ToggleDarkMode key='toggle-theme' />, <NavigationButtons key='nav-buttons' buttons={navButtons} />]}
    />
  );
};

export { SetupHeader };
