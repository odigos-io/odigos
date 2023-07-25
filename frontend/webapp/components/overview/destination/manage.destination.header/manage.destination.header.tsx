import React from "react";
import styled from "styled-components";

const ManageDestinationHeaderWrapper = styled.div`
  display: flex;
  width: 100%;
  height: 104px;
  align-items: center;
  border-radius: 25px;
  margin: 24px 0;
  background: radial-gradient(
      78.09% 72.18% at 100% -0%,
      rgba(150, 242, 255, 0.4) 0%,
      rgba(150, 242, 255, 0) 61.91%
    ),
    linear-gradient(180deg, #2e4c55 0%, #303355 100%);
`;

export function ManageDestinationHeader() {
  return <ManageDestinationHeaderWrapper></ManageDestinationHeaderWrapper>;
}
