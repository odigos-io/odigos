import styled from "styled-components";

export const DestinationContainerWrapper = styled.div`
  height: 100vh;
  /* width: 100%; */
  overflow-y: hidden;
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none; /* IE and Edge */
  scrollbar-width: none; /* Firefox */
`;

export const NewDestinationContainer = styled.div`
  padding: 20px 36px;
`;

export const ManageDestinationWrapper = styled.div`
  padding: 32px;
`;
