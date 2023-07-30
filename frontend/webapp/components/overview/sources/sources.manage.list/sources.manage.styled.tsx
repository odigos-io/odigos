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
    background: var(--dark-mode-dark-1, #07111a81);
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
  width: 100%;
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  overflow-y: scroll;
  padding: 0px 36px;
  padding-bottom: 50px;
  margin-top: 88px;
`;
