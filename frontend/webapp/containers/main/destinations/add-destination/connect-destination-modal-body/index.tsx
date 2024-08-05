import React, { useEffect } from 'react';
import { DestinationTypeItem, StepProps } from '@/types';
import { SideMenu } from '@/components';
import { Divider, SectionTitle } from '@/reuseable-components';
import { Body, Container, SideMenuWrapper } from '../styled';
import { GET_DESTINATION_TYPE_DETAILS } from '@/graphql';
import { useQuery } from '@apollo/client';
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

interface ConnectDestinationModalBodyProps {
  destination: DestinationTypeItem | undefined;
}

export function ConnectDestinationModalBody({
  destination,
}: ConnectDestinationModalBodyProps) {
  const { data } = useQuery(GET_DESTINATION_TYPE_DETAILS, {
    variables: { type: destination?.type },
    skip: !destination,
  });

  useEffect(() => {
    data && console.log({ data });
  }, [data]);

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
      </Body>
    </Container>
  );
}
