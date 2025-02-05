import React from 'react';
import { ROUTES } from '@/utils';
import { useAppStore } from '@/store';
import styled from 'styled-components';
import { SetupHeader } from '@/components';
import { useRouter } from 'next/navigation';
import { useSourceFormData } from '@/hooks';
import { ArrowIcon } from '@odigos/ui-icons';
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

    setAvailableSources(recordedInitialSources);
    setConfiguredSources(getApiSourcesPayload());
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
