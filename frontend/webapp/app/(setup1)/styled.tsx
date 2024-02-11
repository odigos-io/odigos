import styled from 'styled-components';

export const SetupPageContainer = styled.div`
  width: 100vw;
  height: 100vh;

  background: var(
    --gradient-dark,
    radial-gradient(
      44.09% 58.18% at 100% -14%,
      rgba(150, 242, 255, 0.4) 0%,
      rgba(150, 242, 255, 0) 61.91%
    ),
    linear-gradient(181deg, #091824 0%, #2b2f56 100%)
  );
  display: flex;
  flex-direction: column;
  align-items: center;
`;

export const LogoWrapper = styled.div`
  position: absolute;
  top: 20px;
  left: 20px;
`;

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

export const StepListWrapper = styled.div`
  margin: 1% 0 32px 0;
`;
