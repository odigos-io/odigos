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
`;

export const Border = styled.div`
  width: 368px;
  height: 1px;
  margin: 24px 0;
  background: var(--dark-mode-dark-3, #203548);
`;

export const ManagedWrapper = styled.div`
  display: flex;
  padding: 8px 12px;
  align-items: flex-start;
  border-radius: 10px;
  border: 1px solid var(--dark-mode-odigos-torquiz, #96f2ff);
  cursor: pointer;
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
