import React, { useState } from 'react';
import Image from 'next/image';
import { ROUTES } from '@/utils';
import theme from '@/styles/theme';
import { useAppStore } from '@/store';
import styled from 'styled-components';
import { SetupHeader } from '@/components';
import { useRouter } from 'next/navigation';
import { useDestinationCRUD, useSourceCRUD } from '@/hooks';
import { DestinationModal } from '../destination-modal';
import { ConfiguredDestinationsList } from './configured-destinations-list';
import { Button, NotificationNote, SectionTitle, Text } from '@/reuseable-components';

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

const AddDestinationButtonWrapper = styled.div`
  width: 100%;
  margin-top: 24px;
`;

const StyledAddDestinationButton = styled(Button)`
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
`;

export function AddDestinationContainer() {
  const router = useRouter();
  const { createSources, loading: sourcesLoading } = useSourceCRUD();
  const { createDestination, loading: destinationsLoading } = useDestinationCRUD();
  const { configuredSources, configuredFutureApps, configuredDestinations, resetState } = useAppStore((state) => state);

  const [isModalOpen, setModalOpen] = useState(false);
  const handleOpenModal = () => setModalOpen(true);
  const handleCloseModal = () => setModalOpen(false);

  const clickBack = () => {
    router.push(ROUTES.CHOOSE_SOURCES);
  };

  const clickDone = async () => {
    await createSources(configuredSources, configuredFutureApps);
    await Promise.all(configuredDestinations.map(async ({ form }) => await createDestination(form)));

    resetState();
    router.push(ROUTES.OVERVIEW);
  };

  const isSourcesListEmpty = () => !Object.values(configuredSources).some((sources) => !!sources.length);
  const isCreating = sourcesLoading || destinationsLoading;

  return (
    <>
      <HeaderWrapper>
        <SetupHeader
          navigationButtons={[
            {
              label: 'BACK',
              iconSrc: '/icons/common/arrow-white.svg',
              variant: 'secondary',
              onClick: clickBack,
              disabled: isCreating,
            },
            {
              label: 'DONE',
              variant: 'primary',
              onClick: clickDone,
              disabled: isCreating,
            },
          ]}
        />
      </HeaderWrapper>
      <ContentWrapper>
        <SectionTitle title='Configure destinations' description='Select destinations where telemetry data will be sent and configure their settings.' />

        {isSourcesListEmpty() && (
          <NotificationNoteWrapper>
            <NotificationNote
              type='warning'
              message='No sources selected. Please go back to select sources.'
              action={{
                label: 'Select sources',
                onClick: () => router.push(ROUTES.CHOOSE_SOURCES),
              }}
            />
          </NotificationNoteWrapper>
        )}

        <AddDestinationButtonWrapper>
          <StyledAddDestinationButton variant='secondary' onClick={() => handleOpenModal()}>
            <Image src='/icons/common/plus.svg' alt='back' width={16} height={16} />
            <Text color={theme.colors.secondary} size={14} decoration='underline' family='secondary'>
              ADD DESTINATION
            </Text>
          </StyledAddDestinationButton>

          <DestinationModal isOnboarding isOpen={isModalOpen} onClose={handleCloseModal} />
        </AddDestinationButtonWrapper>

        <ConfiguredDestinationsList data={configuredDestinations} />
      </ContentWrapper>
    </>
  );
}
