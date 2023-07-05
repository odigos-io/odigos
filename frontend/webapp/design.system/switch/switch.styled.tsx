import styled from "styled-components";

interface SwitchToggleWrapperProps {
  active: boolean;
}

interface SwitchToggleBtnProps {
  disabled: boolean;
}

export const SwitchInputWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;

export const SwitchToggleWrapper = styled.div<SwitchToggleWrapperProps>`
  position: relative;
  width: 30px;
  height: 16px;
  background-color: ${({ active }) => (active ? "#04dcea" : "#8b92a5")};
  cursor: pointer;
  user-select: none;
  border-radius: 20px;
  padding: 2px;
  display: flex;
  justify-content: center;
  align-items: center;
`;

export const SwitchButtonWrapper = styled.span<SwitchToggleBtnProps>`
  display: flex;
  justify-content: center;
  align-items: center;
  box-sizing: border-box;
  width: 14px;
  height: 14px;
  cursor: pointer;
  color: #fff;
  background-color: ${({ disabled }) => (!disabled ? "#CCD0D2" : "#fff")};
  box-shadow: 0 2px 4px rgb(0, 0, 0, 0.25);
  border-radius: 100%;
  position: absolute;
  transition: all 0.2s ease;
  left: ${({ disabled }) => (!disabled ? 2 : 18)}px;
`;
