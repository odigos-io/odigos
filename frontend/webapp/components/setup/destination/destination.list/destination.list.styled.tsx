import styled from "styled-components";

export const DestinationTypeTitleWrapper = styled.div`
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 24px 0;
`;

export const DestinationListWrapper = styled.div`
  width: 100%;
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none; /* IE and Edge */
  scrollbar-width: none; /* Firefox */
`;

export const EmptyListWrapper = styled.div`
  width: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
`;
