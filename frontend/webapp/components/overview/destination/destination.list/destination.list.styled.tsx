import { styled } from "styled-components";

export const ManagedListWrapper = styled.div`
  width: 100%;
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  overflow-y: scroll;
  padding: 0px 36px;
  padding-bottom: 50px;
`;

export const MenuWrapper = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 36px;
  margin-top: 88px;
`;

export const CardWrapper = styled.div`
  display: flex;
  width: 366px;
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
    background: var(--dark-mode-dark-1, #07111a81);
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
