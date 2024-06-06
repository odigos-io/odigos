import React, { useMemo } from 'react';
import { SETUP } from '@/utils';
import { DestinationType } from '@/types';
import { styled } from 'styled-components';
import FormDangerZone from './form.danger.zone';
import { BackIcon } from '@keyval-dev/design-system';
import { Conditions, KeyvalText } from '@/design.system';
import { CreateConnectionForm, QuickHelp } from '@/components/setup';
import { ManageDestinationHeader } from '../manage.destination.header/manage.destination.header';

interface ManageDestinationProps {
  destinationType: DestinationType;
  selectedDestination: any;
  onBackClick?: () => void;
  onSubmit: (data: any) => void;
  onDelete?: () => void;
}

const BackButtonWrapper = styled.div`
  display: flex;
  align-items: center;
  cursor: pointer;
  p {
    cursor: pointer !important;
  }
`;

const CreateConnectionWrapper = styled.div`
  display: flex;
  gap: 10vw;
`;

export function ManageDestination({
  destinationType,
  selectedDestination,
  onBackClick,
  onSubmit,
  onDelete,
}: ManageDestinationProps) {
  const videoList = useMemo(
    () =>
      destinationType?.fields
        ?.filter((field) => field?.video_url)
        ?.map((field) => ({
          name: field.display_name,
          src: field.video_url,
          thumbnail_url: field.thumbnail_url,
        })),
    [destinationType]
  );

  return (
    <>
      {onBackClick && (
        <BackButtonWrapper onClick={onBackClick}>
          <BackIcon size={14} />
          <KeyvalText size={14}>{SETUP.BACK}</KeyvalText>
        </BackButtonWrapper>
      )}
      <ManageDestinationHeader data={selectedDestination} />
      <CreateConnectionWrapper>
        <div>
          <CreateConnectionForm
            fields={destinationType?.fields}
            destinationNameValue={selectedDestination?.name}
            dynamicFieldsValues={selectedDestination?.fields}
            checkboxValues={selectedDestination?.signals}
            supportedSignals={selectedDestination?.supported_signals}
            onSubmit={(data) => onSubmit(data)}
          />
          {onDelete && (
            <FormDangerZone onDelete={onDelete} data={selectedDestination} />
          )}
        </div>
        <>
          <Conditions conditions={selectedDestination?.conditions} />
          {videoList?.length > 0 && <QuickHelp data={videoList} />}
        </>
      </CreateConnectionWrapper>
    </>
  );
}
