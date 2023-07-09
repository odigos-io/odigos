import styled from "styled-components";

export const SetupSectionContainer = styled.section`
  width: 85%;
  max-width: 1288px;
  height: 1095px;
  border-radius: 24px;
  border: 1px solid var(--dark-mode-dark-4, #374a5b);
  background: var(--dark-mode-dark-2, #132330);
  box-shadow: 0px -6px 16px 0px rgba(0, 0, 0, 0.25),
    4px 4px 16px 0px rgba(71, 231, 241, 0.05),
    -4px 4px 16px 0px rgba(71, 231, 241, 0.05);
`;

export const SetupContentWrapper = styled.div`
  padding: 0 60px;
`;

export const EmptyListWrapper = styled.div`
  width: 100%;
  margin-top: 80px;
  display: flex;
  justify-content: center;
  align-items: center;
`;
