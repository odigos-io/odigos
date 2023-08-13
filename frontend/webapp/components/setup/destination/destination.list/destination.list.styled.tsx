import styled from 'styled-components';

export const DestinationTypeTitleWrapper = styled.div`
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 24px 0;
`;

export const DestinationListWrapper = styled.div<{ repeat: number }>`
  width: 100%;
  display: grid;
  grid-template-columns: ${({ repeat }) => `repeat(${repeat},1fr)`};
  gap: 1vh;
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none;
  scrollbar-width: none;
  @media screen and (max-width: 1700px) {
    grid-template-columns: repeat(4, 1fr);
  }

  @media screen and (max-width: 1500px) {
    grid-template-columns: repeat(3, 1fr);
  }
  @media screen and (max-width: 1250px) {
    grid-template-columns: repeat(2, 1fr);
    padding-bottom: 15vh;
  }
`;

export const EmptyListWrapper = styled.div`
  width: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
`;
