import React, { useState } from 'react';
import { IAppState } from '@/store';
import styled from 'styled-components';
import { useSelector } from 'react-redux';
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

  const sourcesList = useSelector(({ app }) => app.sources);
  const destinations = useSelector(
    ({ app }: { app: IAppState }) => app.configuredDestinationsList
  );
  const isSourcesListEmpty = () => {
    const sourceLen = Object.keys(sourcesList).length === 0;
    if (sourceLen) {
      return true;
    }

    let empty = true;
    for (const source in sourcesList) {
      if (sourcesList[source].length > 0) {
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
              onClick: () => router.back(),
              variant: 'secondary',
            },
            {
              label: 'DONE',
              onClick: () => console.log('Next button clicked'),
              variant: 'primary',
            },
          ]}
        />
      </HeaderWrapper>
      <ContentWrapper>
        <SectionTitle
          title="Configure destinations"
          description="Add backend destinations where collected data will be sent and configure their settings."
        />
        {isSourcesListEmpty() && destinations.length === 0 && (
          <NotificationNoteWrapper>
            <NotificationNote
              type={'warning'}
              text={'No sources selected.'}
              action={{
                label: 'Select sources',
                onClick: () => router.push('/setup/choose-sources'),
              }}
            />
          </NotificationNoteWrapper>
        )}
        <AddDestinationButtonWrapper>
          <AddDestinationButton onClick={() => handleOpenModal()} />
        </AddDestinationButtonWrapper>
        <ConfiguredDestinationsList data={destinations} />
        <AddDestinationModal
          isModalOpen={isModalOpen}
          handleCloseModal={handleCloseModal}
        />
      </ContentWrapper>
    </>
  );
}
