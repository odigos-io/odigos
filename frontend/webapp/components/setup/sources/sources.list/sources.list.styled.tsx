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

export const SourcesListWrapper = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(242px, 1fr));
  gap: 24px;

  height: 56vh; /* Example fixed height */
  overflow-y: auto; /* Enables vertical scrolling if content overflows */
  @media screen and (max-width: 1200px) {
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  }
`;
