'use client';
import React, { useEffect, useRef, useState } from 'react';
import styled from 'styled-components';
import { NodeBaseDataFlow } from './graph';

export const OverviewDataFlowWrapper = styled.div`
  width: 100%;
  height: 100%;
`;

export function OverviewDataFlowContainer() {
  return (
    <OverviewDataFlowWrapper>
      <NodeBaseDataFlow />
    </OverviewDataFlowWrapper>
  );
}
