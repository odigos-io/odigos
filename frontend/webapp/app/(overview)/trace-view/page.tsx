'use client';

import React from 'react';
import { useTraces } from '@/hooks';
import { TraceView } from '@odigos/ui-kit/containers';
import { HEADER_HEIGHT, NO_MENU_GAP_HEIGHT } from '@/utils';

export default function Page() {
  const { traces, isLoading } = useTraces({ serviceName: undefined });

  return (
    <>
      <TraceView heightToRemove={HEADER_HEIGHT + NO_MENU_GAP_HEIGHT} traces={traces} isLoading={isLoading} />
    </>
  );
}
