'use client';
import React from 'react';
import styled from 'styled-components';
import { SETUP } from '@/utils/constants';
import { useRouter } from 'next/navigation';
import { KeyvalText } from '@/design.system';
import { WhiteArrow } from '@/assets/icons/app';

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
      <WhiteArrow />
      <KeyvalText size={14} weight={600}>
        {SETUP.BACK}
      </KeyvalText>
    </BackButtonWrapper>
  );
}
