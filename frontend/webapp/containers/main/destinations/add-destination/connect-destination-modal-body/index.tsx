import React, { useEffect, useMemo, useState } from 'react';
import {
  DestinationDetailsResponse,
  DestinationTypeItem,
  StepProps,
} from '@/types';
import { SideMenu } from '@/components';
import {
  CheckboxList,
  Divider,
  Input,
  SectionTitle,
} from '@/reuseable-components';
import { Body, Container, SideMenuWrapper } from '../styled';
import { GET_DESTINATION_TYPE_DETAILS } from '@/graphql';
import { useQuery } from '@apollo/client';
import styled from 'styled-components';
const SIDE_MENU_DATA: StepProps[] = [
  {
    title: 'DESTINATIONS',
    state: 'finish',
    stepNumber: 1,
  },
  {
    title: 'CONNECTION',
    state: 'active',
    stepNumber: 2,
  },
];

const FormContainer = styled.div`
  display: flex;
  flex-direction: column;
  gap: 24px;
`;

interface ConnectDestinationModalBodyProps {
  destination: DestinationTypeItem | undefined;
}

export function ConnectDestinationModalBody({
  destination,
}: ConnectDestinationModalBodyProps) {
  const { data } = useQuery<DestinationDetailsResponse>(
    GET_DESTINATION_TYPE_DETAILS,
    {
      variables: { type: destination?.type },
      skip: !destination,
    }
  );

  const monitors = useMemo(() => {
    if (!destination) return [];

    const { logs, metrics, traces } = destination.supportedSignals;
    return [
      logs.supported && {
        id: 'logs',
        title: 'Logs',
      },
      metrics.supported && {
        id: 'metrics',
        title: 'Metrics',
      },
      traces.supported && {
        id: 'traces',
        title: 'Traces',
      },
    ].filter(Boolean);
  }, [destination]);

  useEffect(() => {
    data && console.log({ destination, data });
  }, [data]);

  if (!destination) return null;

  return (
    <Container>
      <SideMenuWrapper>
        <SideMenu data={SIDE_MENU_DATA} />
      </SideMenuWrapper>

      <Body>
        <SectionTitle
          title="Create connection"
          description="Connect selected destination with Odigos."
          buttonText="Check connection"
          onButtonClick={() => {}}
        />
        <Divider margin="0 0 24px 0" />
        <FormContainer>
          <CheckboxList
            monitors={monitors as []}
            title="This connection will monitor:"
          />
          <Input
            title="Destination name"
            placeholder="Enter destination name"
          />
        </FormContainer>
      </Body>
    </Container>
  );
}
