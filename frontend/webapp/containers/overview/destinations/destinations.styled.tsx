import styled from 'styled-components';

export const NewDestinationContainer = styled.div`
  height: 81%;
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none;
  scrollbar-width: none;

  @media screen and (max-height: 650px) {
    height: 72%;
  }
  @media screen and (max-height: 550px) {
    height: 65%;
  }
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
