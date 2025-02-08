import styled from 'styled-components';
import { FlexColumn } from '@odigos/ui-components';

export const LayoutContainer = styled(FlexColumn)`
  width: 100%;
  height: 100vh;
  background-color: ${({ theme }) => theme.colors.primary};
  align-items: center;
`;

export const MainContent = styled.div`
  width: 100%;
  height: calc(100vh - 176px);
  position: relative;
`;

export const OnboardingStepperWrapper = styled.div`
  position: absolute;
  left: 24px;
  top: 144px;

  @media (max-width: 1050px) {
    display: none;
  }
`;

// this is to control modal size + scroll
// note: add-destinations does not use this (yet), because it has a custom sidebar
export const ModalBody = styled.div<{ $isNotModal?: boolean }>`
  width: 640px;
  height: ${({ $isNotModal }) => ($isNotModal ? 'fit-content' : 'calc(100vh - 350px)')};
  margin: ${({ $isNotModal }) => ($isNotModal ? '64px 0 0 0' : '64px 7vw 32px 7vw')};
  overflow-y: scroll;
`;
