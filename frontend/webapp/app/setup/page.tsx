'use client';
import { LogoWrapper, SetupPageContainer } from './setup.styled';
import Logo from '@/assets/logos/odigos-gradient.svg';
import { SetupSection } from '@/containers/setup';

export default function SetupPage() {
  return (
    <SetupPageContainer>
      <LogoWrapper>
        <Logo />
      </LogoWrapper>
      <SetupSection />
    </SetupPageContainer>
  );
}
