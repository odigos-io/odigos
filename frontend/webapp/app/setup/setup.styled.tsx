import styled from "styled-components";

export const SetupPageContainer = styled.div`
  width: 100vw;
  height: 100%;
  background: var(
    --gradient-dark,
    radial-gradient(
      78.09% 72.18% at 100% -0%,
      rgba(150, 242, 255, 0.4) 0%,
      rgba(150, 242, 255, 0) 61.91%
    ),
    linear-gradient(180deg, #365963 0%, #2b2f56 100%)
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

export const StepListWrapper = styled.div`
  margin: 94px 0 30px 0;
`;
