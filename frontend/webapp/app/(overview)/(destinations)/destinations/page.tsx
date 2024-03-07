'use client';
import { styled } from 'styled-components';
import { DestinationContainer } from '@/containers/overview';

const DestinationContainerWrapper = styled.div`
  height: 100vh;
  overflow-y: hidden;
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none;
  scrollbar-width: none;
`;

export default function DestinationDashboardPage() {
  return (
    <DestinationContainerWrapper>
      <DestinationContainer />
    </DestinationContainerWrapper>
  );
}
