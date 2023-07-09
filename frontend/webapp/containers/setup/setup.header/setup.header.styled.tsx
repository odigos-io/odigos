import styled from "styled-components";

export const SetupHeaderWrapper = styled.header`
  display: inline-flex;
  padding: 24px 0px;
  align-items: center;
  width: 100%;
  max-width: 1288px;
  justify-content: space-between;
  border-radius: 24px;
  border: 1px solid var(--dark-mode-dark-4, #374a5b);
  background: var(--dark-mode-dark-2, #132330);
`;

export const HeaderTitleWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 24px;
  margin-left: 40px;
`;

export const HeaderButtonWrapper = styled.div`
  display: flex;
  gap: 16px;
  align-items: center;
  margin-right: 40px;
`;
