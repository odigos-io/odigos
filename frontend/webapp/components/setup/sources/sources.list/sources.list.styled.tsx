import styled from "styled-components";

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
  margin: 24px 0;
`;

export const SourcesListWrapper = styled.div`
  width: 100%;
  height: 400px;
  padding-bottom: 300px;
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  overflow-y: scroll;
  scrollbar-width: none;
`;

export const EmptyListWrapper = styled.div`
  width: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
`;
