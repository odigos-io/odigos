'use client';

import React, { type PropsWithChildren } from 'react';
import styled from 'styled-components';
const Layout = styled.section`
  display: flex;
  flex-direction: column;
  flex: 1;
  height: 100%;
  align-self: stretch;
  min-width: 0;
  min-height: 0;
`;
export default function SamplingLayout({ children }: PropsWithChildren) {
  return <Layout>{children}</Layout>;
}
