'use client';

import React from 'react';
import { PlatformType } from '@odigos/ui-kit/types';
import { Settings } from '@odigos/ui-kit/containers';

// Settings is being migrated to consume `useOdigosApi()` directly. Once that
// migration lands, this page becomes a one-line `return <Settings />;`.
export default function Page() {
  return (
    <Settings
      minSupportedVersion={{
        [PlatformType.K8s]: 1.2,
      }}
    />
  );
}
