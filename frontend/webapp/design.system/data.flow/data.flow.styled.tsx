import styled from "styled-components";

export const DataFlowContainer = styled.div`
  width: 100%;
  height: 100%;
`;

export const ControllerWrapper = styled.div`
  button {
    display: flex;
    padding: 8px;
    align-items: center;
    gap: 10px;
    border-radius: 8px;
    border: ${({ theme }) => `1px solid ${theme.colors.blue_grey}`};
    background: #0e1c28 !important;
    margin-bottom: 8px;
  }

  .react-flow__controls button path {
    fill: #fff;
  }
`;
