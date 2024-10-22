import React, { useState } from 'react';
import styled from 'styled-components';
import { useAppStore } from '@/store';
import { K8sActualSource } from '@/types';
import { SetupHeader } from '@/components';
import { useRouter } from 'next/navigation';
import { useConnectSourcesMenuState } from '@/hooks';
import { ChooseSourcesBody } from './choose-sources-body';

const HeaderWrapper = styled.div`
  width: 100vw;
`;
export function ChooseSourcesContainer() {
  const [sourcesList, setSourcesList] = useState<K8sActualSource[]>([]);

  const { setSources, setNamespaceFutureSelectAppsList } = useAppStore();
  const { stateMenu, stateHandlers } = useConnectSourcesMenuState({
    sourcesList,
  });

  const router = useRouter();

  function onNextClick() {
    const { selectedOption, selectedItems, futureAppsCheckbox } = stateMenu;
    if (selectedOption) {
      setSources(selectedItems);
      setNamespaceFutureSelectAppsList(futureAppsCheckbox);
    }
    router.push('/choose-destination');
  }

  return (
    <>
      <HeaderWrapper>
        <SetupHeader
          navigationButtons={[
            {
              label: 'NEXT',
              iconSrc: '/icons/common/arrow-black.svg',
              onClick: () => onNextClick(),
              variant: 'primary',
            },
          ]}
        />
      </HeaderWrapper>
      <ChooseSourcesBody
        stateMenu={stateMenu}
        stateHandlers={stateHandlers}
        sourcesList={sourcesList}
        setSourcesList={setSourcesList}
      />
    </>
  );
}
