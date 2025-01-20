import React, { useEffect, useState } from 'react';
import { ROUTES } from '@/utils';
import { useAppStore } from '@/store';
import styled from 'styled-components';
import { OdigosLogoText } from '@/assets';
import { useRouter } from 'next/navigation';
import { FlexColumn, FlexRow } from '@/styles';
import { useDestinationCRUD, useNamespace, useSourceCRUD } from '@/hooks';
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

export const AwaitPipelineContainer = () => {
  const router = useRouter();
  const { persistSources } = useSourceCRUD();
  const { persistNamespaces } = useNamespace();
  const { createDestination } = useDestinationCRUD();
  const { configuredSources, configuredFutureApps, configuredDestinations, resetState } = useAppStore();

  const [progress, setProgress] = useState(0);

  const doPersist = async () => {
    setProgress(0);
    await persistNamespaces(configuredFutureApps, true);
    setProgress(5);
    await persistSources(configuredSources, true);
    setProgress(10);
    await Promise.all(configuredDestinations.map(async ({ form }) => await createDestination(form)));
    setProgress(15);

    // TODO: await pipeline completion, right now we fake it
    for (let i = 15; i <= 100; i += 5) {
      setTimeout(() => setProgress(i), 500);
    }

    resetState();
    setTimeout(() => router.push(ROUTES.OVERVIEW), 100);
  };

  useEffect(() => {
    doPersist();
  }, []);

  return (
    <Container>
      <OdigosLogoText size={100} />

      <TraceLoader width={400} />

      <TextWrap>
        <FlexRow $gap={16}>
          <Title>Preparing your workspace...</Title>
          <Badge label={`${progress}%`} />
        </FlexRow>

        <Description>It can take up to a few minutes. Grab a cup of coffee, look out a window, and enjoy your free moment!</Description>
      </TextWrap>
    </Container>
  );
};
