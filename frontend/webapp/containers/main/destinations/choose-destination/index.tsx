import React, { useEffect, useState } from 'react';
import { Button, SectionTitle, Text } from '@/reuseable-components';
import styled from 'styled-components';
import Image from 'next/image';
import theme from '@/styles/theme';

const AddDestinationButtonWrapper = styled.div`
  width: 100%;
  margin-top: 24px;
`;

const AddDestinationButton = styled(Button)`
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
`;

export function ChooseDestinationContainer() {
  return (
    <>
      <SectionTitle
        title="Configure destinations"
        description="Add backend destinations where collected data will be sent and configure their settings."
      />
      <AddDestinationButtonWrapper>
        <AddDestinationButton variant="secondary">
          <Image
            src="/icons/common/plus.svg"
            alt="back"
            width={16}
            height={16}
          />
          <Text
            color={theme.colors.secondary}
            size={14}
            decoration={'underline'}
            family="secondary"
          >
            ADD DESTINATION
          </Text>
        </AddDestinationButton>
      </AddDestinationButtonWrapper>
    </>
  );
}
