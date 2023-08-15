import styled from 'styled-components';

export const DestinationListContainer = styled.div`
  width: 100%;
  max-height: 30%;
  height: 400px;
  padding-bottom: 300px;
  margin-top: 2%;
  overflow: scroll;
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none;
  scrollbar-width: none;
  @media screen and (max-height: 700px) {
    height: 40vh;
  }
`;

export const EmptyListWrapper = styled.div`
  width: 100%;
  margin-top: 130px;
  display: flex;
  justify-content: center;
  align-items: center;
`;

export const LoaderWrapper = styled.div`
  height: 500px;
`;
