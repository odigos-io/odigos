import React, { useMemo, useState, type FC, type RefObject } from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import { InstallationStatus } from '@/types';
import { safeJsonParse } from '@odigos/ui-kit/functions';
import { DEFAULT_DATA_STREAM_NAME } from '@odigos/ui-kit/constants';
import { Destination, DestinationFormData } from '@odigos/ui-kit/types';
import { useDataStreamStore, useSetupStore } from '@odigos/ui-kit/store';
import { ArrowLeftIcon, ArrowRightIcon, OdigosLogoText } from '@odigos/ui-kit/icons';
import { useConfig, useDataStreamsCRUD, useDestinationCRUD, useSourceCRUD } from '@/hooks';
import { Header, NavigationButtons, NavigationButtonsProps, Text } from '@odigos/ui-kit/components';
import { type DataStreamSelectionFormRef, ToggleDarkMode, type SourceSelectionFormRef } from '@odigos/ui-kit/containers';

interface SetupHeaderProps {
  step: number;
  streamFormRef?: RefObject<DataStreamSelectionFormRef | null>;
  sourceFormRef?: RefObject<SourceSelectionFormRef | null>;
}

const getFormDataFromDestination = (dest: Destination, selectedStreamName: string): DestinationFormData => {
  const parsedFields = safeJsonParse(dest.fields, {});
  const fieldsArray = Object.entries(parsedFields).map(([key, value]) => ({ key, value: String(value) }));

  const payload: DestinationFormData = {
    type: dest.destinationType.type,
    name: dest.destinationType.displayName,
    disabled: dest.disabled,
    currentStreamName: selectedStreamName,
    exportedSignals: dest.exportedSignals,
    fields: fieldsArray,
  };

  return payload;
};

const firstStep = 2; // The first step in the setup process
const lastStep = 5; // The last step in the setup process

const backRoutes = {
  2: ROUTES.OVERVIEW,
  3: ROUTES.CHOOSE_STREAM,
  4: ROUTES.CHOOSE_SOURCES,
  5: ROUTES.CHOOSE_DESTINATION,
};
const nextRoutes = {
  2: ROUTES.CHOOSE_SOURCES,
  3: ROUTES.CHOOSE_DESTINATION,
  4: ROUTES.SETUP_SUMMARY,
};

const SetupHeader: FC<SetupHeaderProps> = ({ step, streamFormRef, sourceFormRef }) => {
  const router = useRouter();
  const { installationStatus } = useConfig();

  const { persistSources } = useSourceCRUD();
  const { fetchDataStreams } = useDataStreamsCRUD();
  const { createDestination, updateDestination } = useDestinationCRUD();
  const { setSelectedStreamName, selectedStreamName } = useDataStreamStore();
  const { configuredSources, configuredFutureApps, configuredDestinations, configuredDestinationsUpdateOnly, setConfiguredSources, setConfiguredFutureApps, clearStore } = useSetupStore();

  const [isLoading, setIsLoading] = useState(false);

  const onNext = () => {
    if (streamFormRef?.current) {
      const ok = streamFormRef.current.validateForm();
      if (!ok) return;

      const { name } = streamFormRef.current.getFormValues();
      setSelectedStreamName(name);

      // Update the current stream name (in case user changed stream name during the same setup session)
      setConfiguredSources(
        Object.entries(configuredSources).reduce(
          (current, [ns, items]) => {
            current[ns] = items.map((item) => ({
              ...item,
              currentStreamName: name,
            }));

            return current;
          },
          {} as typeof configuredSources,
        ),
      );
    }

    if (sourceFormRef?.current) {
      const { apps, futureApps } = sourceFormRef.current.getFormValues();

      setConfiguredSources(apps);
      setConfiguredFutureApps(futureApps);
    }

    const r = nextRoutes[step as keyof typeof nextRoutes];
    if (r) router.push(r);
  };

  const onBack = () => {
    if (step === firstStep) setSelectedStreamName(DEFAULT_DATA_STREAM_NAME);
    const r = backRoutes[step as keyof typeof backRoutes];
    if (r) router.push(r);
  };

  const onDone = async () => {
    setIsLoading(true);

    await persistSources(configuredSources, configuredFutureApps);

    await Promise.all(
      configuredDestinations.map((dest) => {
        return createDestination(getFormDataFromDestination(dest, selectedStreamName));
      }),
    );

    await Promise.all(
      configuredDestinationsUpdateOnly.map((dest) => {
        return updateDestination(dest.id, getFormDataFromDestination(dest, selectedStreamName));
      }),
    );

    await fetchDataStreams();
    clearStore();
    router.push(ROUTES.OVERVIEW);
  };

  const buttons = useMemo(() => {
    const isNewInstallation = installationStatus === InstallationStatus.New;
    const arr: NavigationButtonsProps['buttons'] = [];

    const nextBtn: NavigationButtonsProps['buttons'][0] = {
      label: 'NEXT',
      icon: () => <ArrowRightIcon />,
      variant: 'primary',
      onClick: onNext,
      disabled: isLoading,
    };
    const backBtn: NavigationButtonsProps['buttons'][0] = {
      label: 'BACK',
      icon: () => <ArrowLeftIcon />,
      variant: 'secondary',
      onClick: onBack,
      disabled: isLoading,
    };
    const doneBtn: NavigationButtonsProps['buttons'][0] = {
      label: 'DONE',
      variant: 'primary',
      onClick: onDone,
      disabled: isLoading,
    };

    if (backRoutes[step as keyof typeof backRoutes] && !(step === 2 && isNewInstallation)) {
      arr.push(backBtn);
    }
    if (nextRoutes[step as keyof typeof nextRoutes]) {
      arr.push(nextBtn);
    }
    if (step === lastStep) {
      arr.push(doneBtn);
    }

    return arr;
  }, [installationStatus, step, isLoading, onNext, onBack, onDone]);

  return (
    <Header
      left={[<OdigosLogoText key='logo' size={150} />]}
      center={[
        <Text key='msg' family='secondary'>
          START WITH ODIGOS
        </Text>,
      ]}
      right={[<ToggleDarkMode key='toggle-theme' />, <NavigationButtons key='nav-buttons' buttons={buttons} />]}
    />
  );
};

export { SetupHeader };
