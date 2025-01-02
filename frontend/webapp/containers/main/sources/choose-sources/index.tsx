import React from 'react';
import { ROUTES } from '@/utils';
import { ArrowIcon } from '@/assets';
import styled from 'styled-components';
import { SetupHeader } from '@/components';
import { useRouter } from 'next/navigation';
import { useSourceFormData } from '@/hooks';
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
    const { recordedInitialValues, getApiPaylod, selectedFutureApps } = menuState;
    const { setAvailableSources, setConfiguredSources, setConfiguredFutureApps } = appState;

    // Types of "recordedInitialValues" and "getApiPaylod()" are actually:
    // { [namespace: string]: Pick<K8sActualSource, 'name' | 'kind' | 'selected' | 'numberOfInstances'>[] };
    //
    // But we will force them as type:
    // { [namespace: string]: K8sActualSource[] };
    //
    // This forced type is to satisfy TypeScript,
    // while knowing that this doesn't break the onboarding flow in any-way...

    setAvailableSources(recordedInitialValues as IAppState['availableSources']);
    setConfiguredSources(getApiPaylod() as IAppState['configuredSources']);
    setConfiguredFutureApps(selectedFutureApps);

    router.push(ROUTES.CHOOSE_DESTINATION);
  };

  return (
    <>
      <HeaderWrapper>
        <SetupHeader
          navigationButtons={[
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
