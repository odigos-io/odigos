import React from 'react';
import { DestinationTypeItem, StepProps } from '@/types';
import { SideMenu } from '@/components';
import { Divider, SectionTitle } from '@/reuseable-components';
import { Body, Container, SideMenuWrapper } from '../styled';
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
  console.log({ destination });
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
