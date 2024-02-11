import React from 'react';
import { SETUP } from '@/utils/constants';
import { Connect } from '@/assets/icons/app';
import { KeyvalText } from '@/design.system';
import { HeaderTitleWrapper, SetupHeaderWrapper } from './styled';

export function ChooseDestinationHeader() {
  const { HEADER } = SETUP;
  return (
    <SetupHeaderWrapper>
      <HeaderTitleWrapper>
        <Connect />
        <KeyvalText>{HEADER.CHOOSE_DESTINATION_TITLE}</KeyvalText>
      </HeaderTitleWrapper>
    </SetupHeaderWrapper>
  );
}
