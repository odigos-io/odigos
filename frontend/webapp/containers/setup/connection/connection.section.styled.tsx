import styled from 'styled-components';

export const CreateConnectionContainer = styled.div`
  display: flex;
  padding: 47px 90px;
  gap: 200px;
  height: 500px;
  overflow: scroll;
  ::-webkit-scrollbar {
    display: none;
  }
  scrollbar-width: none;
  -ms-overflow-style: none;
  @media screen and (max-width: 1400px) {
    height: 330px;
  }
  @media screen and (max-height: 1000px) {
    height: 330px;
  }
`;

export const LoaderWrapper = styled.div`
  height: 500px;
`;
