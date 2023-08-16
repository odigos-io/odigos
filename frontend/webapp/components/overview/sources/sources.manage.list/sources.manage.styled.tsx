import { styled } from 'styled-components';

export const CardWrapper = styled.div`
  display: flex;
  width: auto;
  height: fit-content;
  padding-top: 32px;
  padding-bottom: 24px;
  border-radius: 24px;
  border: 1px solid var(--dark-mode-dark-3, #203548);
  background: var(--dark-mode-dark-1, #07111a);
  align-items: center;
  flex-direction: column;
  gap: 10px;
  cursor: pointer;
  &:hover {
    border: ${({ theme }) => `1px solid  ${theme.colors.secondary}`};
  }
  @media screen and (max-width: 1200px) {
    flex-direction: row;
    padding: 1vw;
    border-radius: 10px;
  }
`;

export const SourceManageContentWrapper = styled.div`
  display: flex;
  align-items: center;
  flex-direction: column;
  gap: 10px;
  div {
    width: fit-content;
  }
  @media screen and (max-width: 1200px) {
    padding-left: 1vw;
    gap: 6px;
    align-items: flex-start;
    p {
      text-align: left !important;
    }
  }
`;

export const EmptyListWrapper = styled.div`
  width: 100%;
  margin-top: 130px;
  display: flex;
  justify-content: center;
  align-items: center;
`;

export const ManagedListWrapper = styled.div`
  display: grid;
  overflow: scroll;
  gap: 24px;
  grid-template-columns: repeat(5, 1fr);
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none;
  scrollbar-width: none;

  @media screen and (max-width: 1600px) {
    grid-template-columns: repeat(4, 1fr);
  }
  @media screen and (max-width: 1500px) {
    grid-template-columns: repeat(3, 1fr);
  }
  @media screen and (max-width: 1200px) {
    grid-template-columns: repeat(2, 1fr);
  }
  @media screen and (max-height: 800px) {
    height: 60vh;
  }
  @media screen and (max-height: 700px) {
    height: 50vh;
  }
`;

export const ManagedContainer = styled.div`
  height: 100%;
  padding: 0px 36px;
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none;
  scrollbar-width: none;
`;
