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
import { Button, FadeLoader, NotificationNote, SectionTitle, Text } from '@/reuseable-components';
import { CenterThis } from '@/styles';

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
  const { createSources } = useSourceCRUD();
  const { createDestination } = useDestinationCRUD();
  const { configuredSources, configuredFutureApps, configuredDestinations, resetState } = useAppStore((state) => state);

  // we need this state, because "loading" from CRUD hooks is a bit delayed, and allows the user to double-click, as well as see elements render in the UI when they should not be rendered.
  const [isLoading, setIsLoading] = useState(false);
  const [isModalOpen, setModalOpen] = useState(false);
  const handleOpenModal = () => setModalOpen(true);
  const handleCloseModal = () => setModalOpen(false);

  const clickBack = () => {
    router.push(ROUTES.CHOOSE_SOURCES);
  };

  const clickDone = async () => {
    setIsLoading(true);

    await createSources(configuredSources, configuredFutureApps);
    await Promise.all(configuredDestinations.map(async ({ form }) => await createDestination(form)));

    resetState();
    router.push(ROUTES.OVERVIEW);
  };

  const isSourcesListEmpty = () => !Object.values(configuredSources).some((sources) => !!sources.length);

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
              disabled: isLoading,
            },
            {
              label: 'DONE',
              variant: 'primary',
              onClick: clickDone,
              disabled: isLoading,
            },
          ]}
        />
      </HeaderWrapper>
      <ContentWrapper>
        <SectionTitle title='Configure destinations' description='Select destinations where telemetry data will be sent and configure their settings.' />

        {!isLoading && isSourcesListEmpty() && (
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
          <StyledAddDestinationButton variant='secondary' disabled={isLoading} onClick={() => handleOpenModal()}>
            <Image src='/icons/common/plus.svg' alt='back' width={16} height={16} />
            <Text color={theme.colors.secondary} size={14} decoration='underline' family='secondary'>
              ADD DESTINATION
            </Text>
          </StyledAddDestinationButton>

          <DestinationModal isOnboarding isOpen={isModalOpen && !isLoading} onClose={handleCloseModal} />
        </AddDestinationButtonWrapper>

        {isLoading ? (
          <CenterThis>
            <FadeLoader style={{ transform: 'scale(2)', marginTop: '3rem' }} />
          </CenterThis>
        ) : (
          <ConfiguredDestinationsList data={configuredDestinations} />
        )}
      </ContentWrapper>
    </>
  );
}
