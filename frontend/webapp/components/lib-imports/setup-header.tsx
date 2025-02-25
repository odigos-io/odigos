import React, { Dispatch, SetStateAction, type FC, type RefObject } from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import Theme from '@odigos/ui-theme';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import { useDestinationCRUD, useSourceCRUD } from '@/hooks';
import { ArrowIcon, OdigosLogoText } from '@odigos/ui-icons';
import { type SourceSelectionFormRef, useSetupStore } from '@odigos/ui-containers';
import { Header, NavigationButtons, NavigationButtonsProps, Text } from '@odigos/ui-components';

interface SetupHeaderProps {
  entityType: ENTITY_TYPES;
  formRef?: RefObject<SourceSelectionFormRef | null>; // in sources
  isLoading?: boolean; // in destinations
  setIsLoading?: Dispatch<SetStateAction<boolean>>; // in destinations
}

const SetupHeader: FC<SetupHeaderProps> = ({ formRef, entityType, isLoading, setIsLoading }) => {
  const router = useRouter();
  const { setAvailableSources, setConfiguredSources, setConfiguredFutureApps } = useSetupStore();

  const { persistSources } = useSourceCRUD();
  const { createDestination } = useDestinationCRUD();
  const { configuredSources, configuredFutureApps, configuredDestinations, resetState } = useSetupStore();

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
    entityType === ENTITY_TYPES.SOURCE
      ? [
          {
            label: 'NEXT',
            icon: ArrowIcon,
            onClick: () => onNext(),
            variant: 'primary',
          },
        ]
      : entityType === ENTITY_TYPES.DESTINATION
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
      right={[<Theme.ToggleDarkMode key='toggle-theme' />, <NavigationButtons key='nav-buttons' buttons={navButtons} />]}
    />
  );
};

export { SetupHeader };
