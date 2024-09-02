import React from 'react';
import {
  LibCheckboxWrapper,
  LibNameWrapper,
  StyledLibraryOptionContainer,
} from './style.styled';

import { KeyvalCheckbox, KeyvalText } from '@/design.system';
import theme from '@/styles/palette';

const SOURCES = {
  LANGUAGE: 'Language',
  LIBRARY: 'Library',
};

export default function InstrumentedLibraryOption({
  name,
  selected,
  onChange,
  disabled = false,
}: {
  name: string;
  selected: boolean;
  onChange: (name: string) => void;
  disabled: boolean;
}) {
  return (
    <StyledLibraryOptionContainer style={{ opacity: disabled ? 0.5 : 1 }}>
      <LibCheckboxWrapper disabled={disabled}>
        <KeyvalCheckbox
          value={selected}
          onChange={() => onChange(name)}
          disabled={disabled}
        />
      </LibCheckboxWrapper>
      <LibNameWrapper>
        <KeyvalText size={12} color={theme.text.light_grey}>
          {SOURCES.LIBRARY}
        </KeyvalText>
        <KeyvalText size={14} weight={600}>
          {name}
        </KeyvalText>
      </LibNameWrapper>
    </StyledLibraryOptionContainer>
  );
}
