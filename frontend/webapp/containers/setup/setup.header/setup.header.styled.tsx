import styled from "styled-components";

export const SetupHeaderWrapper = styled.header`
  display: inline-flex;
  padding: 24px 40px;
  align-items: center;
  width: 1208px;
  justify-content: space-between;
  border-radius: 24px;
  border: 1px solid var(--dark-mode-dark-4, #374a5b);
  background: var(--dark-mode-dark-2, #132330);
`;

export const HeaderTitleWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 24px;
`;

export const HeaderButtonWrapper = styled.div`
  display: flex;
  gap: 16px;
  align-items: center;
`;
