'use client';

import React from 'react';
import { useTraces } from '@/hooks';
import { TraceView } from '@odigos/ui-kit/containers';
import { HEADER_HEIGHT, MENU_BAR_HEIGHT } from '@/utils';

export default function Page() {
  const { traces } = useTraces({ serviceName: undefined });
  console.log('useTraces', traces);

  return (
    <>
      <TraceView heightToRemove={HEADER_HEIGHT + MENU_BAR_HEIGHT} traces={traces} />
    </>
  );
}
