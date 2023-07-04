import styled from "styled-components";

export const RadioButtonContainer = styled.label`
  width: 24px;
  height: 24px;
  color: #303030;
  font-size: 14px;
  font-weight: 400;
  margin-right: 7px;
  -webkit-tap-highlight-color: transparent;
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
`;

export const RadioButtonBorder = styled.span`
  cursor: pointer;
  width: 23px;
  height: 23px;
  border: 2px solid #ccd0d2;
  border-radius: 50%;
  display: inline-block;
  position: relative;
`;
