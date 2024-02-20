import styled from 'styled-components';
import { NewDestinationList } from '@/containers/overview';

export const PageContainer = styled.div`
  height: 100vh;
`;

export default function CreateDestinationPage() {
  return (
    <PageContainer>
      <NewDestinationList />
    </PageContainer>
  );
}
