import React, { useState } from 'react';
import { ROUTES } from '@/utils';
import { useAppStore } from '@/store';
import styled from 'styled-components';
import { useRouter } from 'next/navigation';
import { AddDestinationModal } from './add-destination-modal';
import { AddDestinationButton, SetupHeader } from '@/components';
import { NotificationNote, SectionTitle } from '@/reuseable-components';
import { ConfiguredDestinationsList } from './configured-destinations-list';

const AddDestinationButtonWrapper = styled.div`
  width: 100%;
  margin-top: 24px;
`;

const ContentWrapper = styled.div`
  width: 640px;
  padding-top: 64px;
`;

const HeaderWrapper = styled.div`
  width: 100vw;
`;

const NotificationNoteWrapper = styled.div`
  margin-top: 24px;
`;

export function ChooseDestinationContainer() {
  const [isModalOpen, setModalOpen] = useState(false);

  const router = useRouter();
  const { configuredSources, configuredDestinations, resetState } = useAppStore((state) => state);

  const isSourcesListEmpty = () => {
    const sourceLen = Object.keys(configuredSources).length === 0;
    if (sourceLen) {
      return true;
    }

    let empty = true;
    for (const source in configuredSources) {
      if (configuredSources[source].length > 0) {
        empty = false;
        break;
      }
    }
    return empty;
  };

  const handleOpenModal = () => setModalOpen(true);
  const handleCloseModal = () => setModalOpen(false);

  return (
    <>
      <HeaderWrapper>
        <SetupHeader
          navigationButtons={[
            {
              label: 'BACK',
              iconSrc: '/icons/common/arrow-white.svg',
              onClick: () => router.push(ROUTES.CHOOSE_SOURCES),
              variant: 'secondary',
            },
            {
              label: 'DONE',
              onClick: () => {
                resetState();
                router.push(ROUTES.OVERVIEW);
              },
              variant: 'primary',
            },
          ]}
        />
      </HeaderWrapper>
      <ContentWrapper>
        <SectionTitle title='Configure destinations' description='Select destinations where telemetry data will be sent and configure their settings.' />
        {isSourcesListEmpty() && configuredDestinations.length === 0 && (
          <NotificationNoteWrapper>
            <NotificationNote
              type={'warning'}
              message={'No sources selected. Please go back to select sources.'}
              action={{
                label: 'Select sources',
                onClick: () => router.push(ROUTES.CHOOSE_SOURCES),
              }}
            />
          </NotificationNoteWrapper>
        )}
        <AddDestinationButtonWrapper>
          <AddDestinationButton onClick={() => handleOpenModal()} />
        </AddDestinationButtonWrapper>
        <ConfiguredDestinationsList data={configuredDestinations} />
        <AddDestinationModal isOpen={isModalOpen} onClose={handleCloseModal} />
      </ContentWrapper>
    </>
  );
}
