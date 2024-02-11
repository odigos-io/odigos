'use client';
import React from 'react';
import Logo from '@/assets/logos/odigos-gradient.svg';
import { LogoWrapper, SetupPageContainer } from './styled';

export default function SetupLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <SetupPageContainer>
      <LogoWrapper>
        <Logo />
      </LogoWrapper>
      {children}
    </SetupPageContainer>
  );
}
