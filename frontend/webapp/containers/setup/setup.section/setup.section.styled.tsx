import styled from "styled-components";

export const SetupSectionContainer = styled.section`
  position: relative;
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
  height: 100%;
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none; /* IE and Edge */
  scrollbar-width: none; /* Firefox */
`;

export const StepListWrapper = styled.div`
  margin: 94px 0 32px 0;
`;

export const BackButtonWrapper = styled.div`
  position: absolute;
  display: flex;
  align-items: center;
  gap: 10px;
  top: -54px;
  cursor: pointer;
  p {
    cursor: pointer !important;
  }
`;
