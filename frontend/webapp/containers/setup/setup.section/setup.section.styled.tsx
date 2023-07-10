import styled from "styled-components";

export const SetupSectionContainer = styled.section`
  width: 85%;
  max-width: 1288px;
  height: 1095px;
  border-radius: 24px;
  border: ${({ theme }) => `1px solid ${theme.colors.blue_grey}`};
  background: ${({ theme }) => theme.colors.light_dark};
  box-shadow: 0px -6px 16px 0px rgba(0, 0, 0, 0.25),
    4px 4px 16px 0px rgba(71, 231, 241, 0.05),
    -4px 4px 16px 0px rgba(71, 231, 241, 0.05);
`;

export const SetupContentWrapper = styled.div`
  padding: 0 60px;
`;
