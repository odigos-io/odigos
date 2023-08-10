import { styled } from "styled-components";

export const CardWrapper = styled.div`
  display: flex;
  width: 272px;
  height: 152px;
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
  gap: 24px;
  overflow: scroll;
  grid-template-columns: repeat(5, 1fr);
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none;
  scrollbar-width: none;

  @media screen and (max-width: 1800px) {
    grid-template-columns: repeat(4, 1fr);
  }
  @media screen and (max-width: 1500px) {
    grid-template-columns: repeat(3, 1fr);
    height: 680px;
  }
  @media screen and (max-width: 1200px) {
    grid-template-columns: repeat(2, 1fr);
    height: 650px;
  }
`;

export const ManagedContainer = styled.div`
  width: 100%;
  height: 100%;
  padding: 0px 36px;
`;
