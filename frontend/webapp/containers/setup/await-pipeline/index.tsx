import React from 'react';
import { FlexColumn, FlexRow } from '@/styles';
import { OdigosLogoText } from '@/assets';
import styled, { useTheme } from 'styled-components';
import { Badge, Text, TraceLoader } from '@/reuseable-components';

const Container = styled(FlexColumn)`
  width: 100vw;
  height: 100vh;
  align-items: center;
  justify-content: center;
`;

const TextWrap = styled(FlexColumn)`
  max-width: 400px;
  gap: 12px;
  align-items: center;
  justify-content: center;
`;

export const AwaitPipelineContainer = () => {
  const theme = useTheme();

  return (
    <Container>
      <OdigosLogoText size={80} />

      <TraceLoader width={400} />

      <TextWrap>
        <FlexRow $gap={16}>
          <Text align='center' size={24}>
            Preparing your workspace...
          </Text>
          <Badge label={`${69}%`} />
        </FlexRow>

        <Text align='center' size={18} color={theme.text.info}>
          It can take up to a few minutes. Grab a cupof coffee, look out a window, and enjoyyour free moment!
        </Text>
      </TextWrap>
    </Container>
  );
};
