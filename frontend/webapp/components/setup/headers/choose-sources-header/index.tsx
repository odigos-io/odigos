import React from 'react';
import { SETUP } from '@/utils';
import theme from '@/styles/palette';
import { KeyvalButton, KeyvalText } from '@/design.system';
import { ChargeIcon, RightArrowIcon } from '@keyval-dev/design-system';
import {
  HeaderButtonWrapper,
  HeaderTitleWrapper,
  SetupHeaderWrapper,
  TotalSelectedWrapper,
} from './styled';

type SetupHeaderProps = {
  onNextClick: () => void;
  totalSelected: number;
};

export function ChooseSourcesHeader({
  onNextClick,
  totalSelected,
}: SetupHeaderProps) {
  const { HEADER } = SETUP;
  return (
    <SetupHeaderWrapper>
      <HeaderTitleWrapper>
        <ChargeIcon />
        <KeyvalText>{HEADER.CHOOSE_SOURCE_TITLE}</KeyvalText>
      </HeaderTitleWrapper>
      <HeaderButtonWrapper>
        {!isNaN(totalSelected) && (
          <TotalSelectedWrapper>
            <KeyvalText>{totalSelected}</KeyvalText>
            <KeyvalText>{SETUP.SELECTED}</KeyvalText>
          </TotalSelectedWrapper>
        )}

        <KeyvalButton data-cy={'choose-source-next-click'} onClick={onNextClick} style={{ gap: 10, width: 120 }}>
          <KeyvalText size={20} weight={600} color={theme.text.dark_button}>
            {SETUP.NEXT}
          </KeyvalText>
          <RightArrowIcon />
        </KeyvalButton>
      </HeaderButtonWrapper>
    </SetupHeaderWrapper>
  );
}
