import { styled } from "styled-components";

export const ManagedListWrapper = styled.div`
  display: grid;
  grid-gap: 24px;

  padding: 0px 36px;
  padding-bottom: 50px;
  grid-template-columns: repeat(4, 1fr);
  overflow-y: scroll;
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none;
  scrollbar-width: none;

  @media screen and (max-width: 1850px) {
    grid-template-columns: repeat(3, 1fr);
  }
  @media screen and (max-width: 1450px) {
    grid-template-columns: repeat(2, 1fr);
    height: 75%;
  }
`;

export const MenuWrapper = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 36px;
`;

export const CardWrapper = styled.div`
  display: flex;
  width: 366px;
  height: 190px;
  padding-top: 32px;
  padding-bottom: 24px;
  flex-direction: column;
  align-items: center;
  border-radius: 24px;
  border: 1px solid var(--dark-mode-dark-3, #203548);
  background: var(--dark-mode-dark-1, #07111a);
  display: flex;
  align-items: center;
  flex-direction: column;
  cursor: pointer;
  &:hover {
    border: ${({ theme }) => `1px solid  ${theme.colors.secondary}`};
  }
`;

export const ApplicationNameWrapper = styled.div`
  display: inline-block;
  text-overflow: ellipsis;
  max-width: 224px;
  height: 56px;
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 12px;
  margin-bottom: 20px;
`;

export const EmptyListWrapper = styled.div`
  width: 100%;
  margin-top: 130px;
  display: flex;
  justify-content: center;
  align-items: center;
`;
