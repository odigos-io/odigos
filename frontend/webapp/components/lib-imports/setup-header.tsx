import React, { useState, type FC, type RefObject } from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import { safeJsonParse } from '@odigos/ui-kit/functions';
import { ArrowIcon, OdigosLogoText } from '@odigos/ui-kit/icons';
import { Destination, DestinationFormData } from '@odigos/ui-kit/types';
import { useDataStreamStore, useSetupStore } from '@odigos/ui-kit/store';
import { useDataStreamsCRUD, useDestinationCRUD, useSourceCRUD } from '@/hooks';
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
    currentStreamName: selectedStreamName,
    exportedSignals: dest.exportedSignals,
    fields: fieldsArray,
  };

  return payload;
};

const backRoutes = {
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

  const { persistSources } = useSourceCRUD();
  const { fetchDataStreams } = useDataStreamsCRUD();
  const { createDestination, updateDestination } = useDestinationCRUD();
  const { setSelectedStreamName, selectedStreamName } = useDataStreamStore();
  const { configuredSources, configuredFutureApps, configuredDestinations, configuredDestinationsUpdateOnly, setAvailableSources, setConfiguredSources, setConfiguredFutureApps, resetState } =
    useSetupStore();

  const [isLoading, setIsLoading] = useState(false);

  const onNext = () => {
    if (streamFormRef?.current) {
      // const ok = streamFormRef.current.validateForm();
      // if (!ok) return;

      const { name } = streamFormRef.current.getFormValues();
      setSelectedStreamName(name);
    }

    if (sourceFormRef?.current) {
      const { initial, apps, futureApps } = sourceFormRef.current.getFormValues();

      setAvailableSources(initial);
      setConfiguredSources(apps);
      setConfiguredFutureApps(futureApps);
    }

    const r = nextRoutes[step as keyof typeof nextRoutes];
    if (r) router.push(r);
  };

  const onBack = () => {
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
    resetState();
    router.push(ROUTES.OVERVIEW);
  };

  const nextBtn: NavigationButtonsProps['buttons'][0] = {
    label: 'NEXT',
    icon: ArrowIcon,
    variant: 'primary',
    onClick: onNext,
    disabled: isLoading,
  };
  const backBtn: NavigationButtonsProps['buttons'][0] = {
    label: 'BACK',
    icon: ArrowIcon,
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

  const buttons = step === 2 ? [nextBtn] : step === 5 ? [backBtn, doneBtn] : [backBtn, nextBtn];

  return (
    <Header
      left={[<OdigosLogoText key='logo' size={100} />]}
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
