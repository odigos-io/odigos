'use client';
import React from 'react';
import { SETUP } from '@/utils';
import styled from 'styled-components';
import { KeyvalText } from '@/design.system';
import { WhiteArrowIcon } from '@keyval-dev/design-system';

const BackButtonWrapper = styled.div`
  position: absolute;
  width: 100%;
  display: flex;
  align-items: center;
  gap: 10px;
  top: -34px;
  cursor: pointer;
  p {
    cursor: pointer !important;
  }
`;

interface SetupBackButtonProps {
  onBackClick: () => void;
}

export function SetupBackButton({ onBackClick }: SetupBackButtonProps) {
  return (
    <BackButtonWrapper onClick={onBackClick}>
      <WhiteArrowIcon />
      <KeyvalText size={14} weight={600}>
        {SETUP.BACK}
      </KeyvalText>
    </BackButtonWrapper>
  );
}
