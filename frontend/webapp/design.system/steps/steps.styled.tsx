import styled from "styled-components";

interface DisabledProp {
  disabled: boolean;
}
export const StepsContainer = styled.div`
  display: flex;
`;

export const StepItemWrapper = styled.div`
  display: flex;
  align-items: center;
`;

export const FloatingBoxTextWrapper = styled.div<DisabledProp>`
  opacity: ${({ disabled }) => (disabled ? "0.4" : "1")};
`;

export const StepItemTextWrapper = styled(FloatingBoxTextWrapper)`
  margin: 0 8px;
`;

export const StepItemBorder = styled.div`
  width: 54px;
  height: 1px;
  background-color: #8b92a5;
  margin-top: 2px;
  margin-right: 8px;
`;
