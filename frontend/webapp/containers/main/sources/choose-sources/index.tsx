import React from 'react';
import { ROUTES } from '@/utils';
import { useAppStore } from '@/store';
import styled from 'styled-components';
import { SetupHeader } from '@/components';
import { useRouter } from 'next/navigation';
import { useSourceFormData } from '@/hooks';
import { ChooseSourcesBody } from './choose-sources-body';

const HeaderWrapper = styled.div`
  width: 100vw;
`;

export function ChooseSourcesContainer() {
  const router = useRouter();
  const menuState = useSourceFormData();
  const { setAvailableSources, setConfiguredSources, setConfiguredFutureApps } = useAppStore();

  const onNext = () => {
    const { selectedNamespace, availableSources, selectedSources, selectedFutureApps } = menuState;

    if (selectedNamespace) {
      setAvailableSources(availableSources);
      setConfiguredSources(selectedSources);
      setConfiguredFutureApps(selectedFutureApps);
    }

    router.push(ROUTES.CHOOSE_DESTINATION);
  };

  return (
    <>
      <HeaderWrapper>
        <SetupHeader
          navigationButtons={[
            {
              label: 'NEXT',
              iconSrc: '/icons/common/arrow-black.svg',
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
