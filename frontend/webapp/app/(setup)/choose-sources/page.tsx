'use client';

import React, { useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import Theme from '@odigos/ui-theme';
import { useNamespace, useSSE } from '@/hooks';
import { OnboardingStepperWrapper } from '@/styles';
import { ArrowIcon, OdigosLogoText } from '@odigos/ui-icons';
import { FormRef, SourceSelectionForm, useSetupStore } from '@odigos/ui-containers';
import { Header, NavigationButtons, Stepper, Text } from '@odigos/ui-components';

export default function Page() {
  // call important hooks that should run on page-mount
  useSSE();

  const router = useRouter();
  const setupState = useSetupStore();

  const [selectedNamespace, setSelectedNamespace] = useState('');
  const onSelectNamespace = (ns: string) => setSelectedNamespace((prev) => (prev === ns ? '' : ns));
  const { allNamespaces, data: namespace, loading: nsLoad } = useNamespace(selectedNamespace);

  const onNext = () => {
    if (formRef.current) {
      // const { initial, apps, futureApps } = formRef.current.getFormValues();
      const { apps, futureApps } = formRef.current.getFormValues();
      const { setAvailableSources, setConfiguredSources, setConfiguredFutureApps } = setupState;

      setAvailableSources({});
      setConfiguredSources(apps);
      setConfiguredFutureApps(futureApps);

      router.push(ROUTES.CHOOSE_DESTINATION);
    }
  };

  const formRef = useRef<FormRef>(null);

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
                label: 'NEXT',
                icon: ArrowIcon,
                onClick: () => onNext(),
                variant: 'primary',
              },
            ]}
          />,
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

      <SourceSelectionForm
        ref={formRef}
        componentType='FAST'
        isModal={false}
        namespaces={allNamespaces}
        namespace={namespace}
        namespacesLoading={nsLoad}
        selectedNamespace={selectedNamespace}
        onSelectNamespace={onSelectNamespace}
      />
    </>
  );
}
