import styled from 'styled-components';

export const CreateConnectionContainer = styled.div`
  display: flex;
  padding: 47px 90px;
  gap: 10vw;
  overflow: scroll;
  ::-webkit-scrollbar {
    display: none;
  }
  scrollbar-width: none;
  -ms-overflow-style: none;
  height: 80%;
  max-height: 600px;

  @media screen and (max-height: 750px) {
    height: 50vh;
  }
`;

export const LoaderWrapper = styled.div`
  height: 500px;
`;
