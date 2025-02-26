'use client';

import React from 'react';
import { useSourceCRUD } from '@/hooks';
import type { Source } from '@odigos/ui-utils';
import { MultiSourceControl, type SourceSelectionFormData, SourceTable } from '@odigos/ui-containers';

export default function Page() {
  const { sources, persistSources } = useSourceCRUD();

  return (
    <>
      <SourceTable sources={sources} tableMaxHeight='calc(100vh - 220px)' />

      <MultiSourceControl
        totalSourceCount={sources.length}
        uninstrumentSources={(payload) => {
          const inp: SourceSelectionFormData = {};

          Object.entries(payload).forEach(([namespace, sources]: [string, Source[]]) => {
            inp[namespace] = sources.map(({ name, kind }) => ({ name, kind, selected: false }));
          });

          persistSources(inp, {});
        }}
      />
    </>
  );
}
