'use client';
import React from 'react';
import styled from 'styled-components';
import { OdigosLogoText } from '@/assets';
import { FlexColumn, FlexRow } from '@/styles';
import { Badge, Text, TraceLoader } from '@/reuseable-components';

const Container = styled(FlexColumn)`
  width: 100vw;
  height: 100vh;
  gap: 64px;
  align-items: center;
  justify-content: center;
`;

const TextWrap = styled(FlexColumn)`
  max-width: 400px;
  gap: 12px;
  align-items: center;
  justify-content: center;
`;

const Title = styled(Text)`
  text-align: center;
  font-size: 24px;
`;

const Description = styled(Text)`
  text-align: center;
  font-size: 18px;
  color: ${({ theme }) => theme.text.info};
  line-height: 26px;
`;

const AwaitPipelineContainer = () => {
  return (
    <Container>
      <OdigosLogoText size={100} />

      <TraceLoader width={400} />

      <TextWrap>
        <FlexRow $gap={16}>
          <Title>Preparing your workspace...</Title>
          <Badge label={`${69}%`} />
        </FlexRow>

        <Description>It can take up to a few minutes. Grab a cup of coffee, look out a window, and enjoy your free moment!</Description>
      </TextWrap>
    </Container>
  );
};

// use "default" so we can import it "dynamically" in /app/(setup)/await-pipeline/page.tsx
export default AwaitPipelineContainer;
