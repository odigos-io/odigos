import React, { useState } from 'react';
import { ROUTES } from '@/utils';
import theme from '@/styles/theme';
import { useAppStore } from '@/store';
import styled from 'styled-components';
import { SetupHeader } from '@/components';
import { useRouter } from 'next/navigation';
import { NOTIFICATION_TYPE } from '@/types';
import { ArrowIcon, PlusIcon } from '@/assets';
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

export const AddDestinationContainer = () => {
  const router = useRouter();
  const { configuredSources, configuredDestinations } = useAppStore((state) => state);

  const [isModalOpen, setModalOpen] = useState(false);
  const handleOpenModal = () => setModalOpen(true);
  const handleCloseModal = () => setModalOpen(false);

  const clickBack = () => router.push(ROUTES.CHOOSE_SOURCES);
  const clickDone = async () => router.push(ROUTES.AWAIT_PIPELINE);
  const isSourcesListEmpty = () => !Object.values(configuredSources).some((sources) => !!sources.length);

  return (
    <>
      <HeaderWrapper>
        <SetupHeader
          navigationButtons={[
            {
              label: 'BACK',
              icon: ArrowIcon,
              variant: 'secondary',
              onClick: clickBack,
            },
            {
              label: 'DONE',
              variant: 'primary',
              onClick: clickDone,
            },
          ]}
        />
      </HeaderWrapper>
      <ContentWrapper>
        <SectionTitle title='Configure destinations' description='Select destinations where telemetry data will be sent and configure their settings.' />

        {isSourcesListEmpty() && (
          <NotificationNoteWrapper>
            <NotificationNote type={NOTIFICATION_TYPE.WARNING} message='No sources selected. Please go back to select sources.' action={{ label: 'Select sources', onClick: clickBack }} />
          </NotificationNoteWrapper>
        )}

        <AddDestinationButtonWrapper>
          <StyledAddDestinationButton variant='secondary' onClick={() => handleOpenModal()}>
            <PlusIcon />
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
};
