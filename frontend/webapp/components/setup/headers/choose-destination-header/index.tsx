import React from 'react';
import { SETUP } from '@/utils';
import theme from '@/styles/palette';
import { Connect } from '@/assets/icons/app';
import { KeyvalText } from '@/design.system';
import {
  HeaderTitleWrapper,
  SetupHeaderWrapper,
  TotalSelectedWrapper,
} from './styled';

interface ChooseDestinationHeaderProps {
  totalSelectedApps: number;
}

export function ChooseDestinationHeader({
  totalSelectedApps,
}: ChooseDestinationHeaderProps) {
  const { HEADER } = SETUP;

  return (
    <SetupHeaderWrapper>
      <HeaderTitleWrapper>
        <Connect />
        <KeyvalText>{HEADER.CHOOSE_DESTINATION_TITLE}</KeyvalText>
      </HeaderTitleWrapper>
      <TotalSelectedWrapper>
        {totalSelectedApps ? (
          <>
            <KeyvalText>{totalSelectedApps}</KeyvalText>
            <KeyvalText>{SETUP.SOURCE_SELECTED}</KeyvalText>
          </>
        ) : (
          <>
            <KeyvalText color={theme.colors.orange_brown}>
              {SETUP.NONE_SOURCE_SELECTED}
            </KeyvalText>
          </>
        )}
      </TotalSelectedWrapper>
    </SetupHeaderWrapper>
  );
}
