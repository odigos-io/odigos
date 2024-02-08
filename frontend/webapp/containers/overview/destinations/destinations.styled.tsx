import styled from 'styled-components';

export const NewDestinationContainer = styled.div`
  padding: 0px 36px;
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none;
  scrollbar-width: none;
`;

export const ManageDestinationWrapper = styled.div`
  padding: 32px;
  overflow-y: scroll;
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none;
  scrollbar-width: none;
  height: 80vh;
  @media screen and (max-height: 750px) {
    height: 85vh;
  }
`;
