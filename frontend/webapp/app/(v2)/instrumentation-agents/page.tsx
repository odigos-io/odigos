'use client';

import React from 'react';
import { InstrumentationAgents } from '@odigos/ui-kit/containers';

// Agents and Go offsets both flow via `useOdigosApi()` inside the kit container.
export default function Page() {
  return <InstrumentationAgents />;
}
