import React from 'react';
import { ROUTES } from '@/utils';
import styled from 'styled-components';
import { SetupHeader } from '@/components';
import { useRouter } from 'next/navigation';
import { useSourceFormData } from '@/hooks';
import { ArrowIcon } from '@odigos/ui-icons';
import { IAppState, useAppStore } from '@/store';
import { ChooseSourcesBody } from './choose-sources-body';

const HeaderWrapper = styled.div`
  width: 100vw;
`;

export function ChooseSourcesContainer() {
  const router = useRouter();
  const appState = useAppStore();
  const menuState = useSourceFormData();

  const onNext = () => {
    const { recordedInitialSources, getApiSourcesPayload, getApiFutureAppsPayload } = menuState;
    const { setAvailableSources, setConfiguredSources, setConfiguredFutureApps } = appState;

    // Types of "recordedInitialSources" and "getApiSourcesPayload()" are actually:
    // { [namespace: string]: Pick<K8sActualSource, 'name' | 'kind' | 'selected' | 'numberOfInstances'>[] };
    //
    // But we will force them as type:
    // { [namespace: string]: K8sActualSource[] };
    //
    // This forced type is to satisfy TypeScript,
    // while knowing that this doesn't break the onboarding flow in any-way...

    setAvailableSources(recordedInitialSources as unknown as IAppState['availableSources']);
    setConfiguredSources(getApiSourcesPayload() as unknown as IAppState['configuredSources']);
    setConfiguredFutureApps(getApiFutureAppsPayload());

    router.push(ROUTES.CHOOSE_DESTINATION);
  };

  return (
    <>
      <HeaderWrapper>
        <SetupHeader
          buttons={[
            {
              label: 'NEXT',
              icon: ArrowIcon,
              onClick: () => onNext(),
              variant: 'primary',
            },
          ]}
        />
      </HeaderWrapper>
      <ChooseSourcesBody componentType='FAST' {...menuState} />
    </>
  );
}
