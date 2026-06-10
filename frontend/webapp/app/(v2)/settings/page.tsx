'use client';

import React from 'react';
import { Settings } from '@odigos/ui-kit/containers/v2';

// Settings is being migrated to consume `useOdigosApi()` directly. Once that
// migration lands, this page becomes a one-line `return <Settings />;`.
export default function Page() {
  return <Settings minSupportedVersion={1.2} />;
}
