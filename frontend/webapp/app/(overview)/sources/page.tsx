'use client';

import React from 'react';
import { useSourceCRUD } from '@/hooks';
import { MultiSourceControl, SourceTable } from '@odigos/ui-containers';

export default function Page() {
  const { sources, persistSources } = useSourceCRUD();

  return (
    <>
      <SourceTable sources={sources} tableMaxHeight='calc(100vh - 220px)' />
      <MultiSourceControl totalSourceCount={sources.length} uninstrumentSources={(payload) => persistSources(payload, {})} />
    </>
  );
}
