import styled from 'styled-components';

export const SourcesListContainer = styled.div`
  width: 100%;
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none; /* IE and Edge */
  scrollbar-width: none; /* Firefox */
`;

export const SourcesTitleWrapper = styled.div`
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 2% 0;
`;

export const SourcesListWrapper = styled.div<{ repeat: number }>`
  width: 100%;
  height: 400px;
  padding-bottom: 300px;
  gap: 1vh;
  overflow-y: scroll;
  scrollbar-width: none;
  -ms-overflow-style: none;
  display: grid;
  grid-template-columns: ${({ repeat }) => `repeat(${repeat},1fr)`};

  ::-webkit-scrollbar {
    display: none;
  }

  @media screen and (max-width: 1750px) {
    grid-template-columns: repeat(4, 1fr);
  }

  @media screen and (max-width: 1500px) {
    grid-template-columns: repeat(3, 1fr);
    height: 300px;
  }
  @media screen and (max-width: 1150px) {
    grid-template-columns: repeat(2, 1fr);
    height: 200px;
  }
`;
