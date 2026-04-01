'use client';

import { createElement, useEffect, type ReactElement } from 'react';
import type { Source } from '@odigos/ui-kit/types';
import { SourceProfilerTab } from './SourceProfilerTab';

declare global {
  interface Window {
    /** Set by patched @odigos/ui-kit SourceDrawer when the "Profiler" tab is selected. */
    __ODIGOS_SOURCE_PROFILER_TAB__?: (source: Source) => ReactElement;
  }
}

function profilerTabRenderer(source: Source): ReactElement {
  return createElement(SourceProfilerTab, {
    key: `${source.namespace}:${String(source.kind ?? 'Deployment')}:${source.name}`,
    source,
  });
}

/**
 * Registers the profiler tab renderer for the Source drawer (ui-kit patch).
 * Registration runs on every render so window.__ODIGOS_SOURCE_PROFILER_TAB__ exists before
 * the drawer reads it (useEffect-only registration caused a first-paint race).
 */
export function ProfilerTabBridge() {
  if (typeof window !== 'undefined') {
    window.__ODIGOS_SOURCE_PROFILER_TAB__ = profilerTabRenderer;
  }
  useEffect(() => {
    return () => {
      if (typeof window !== 'undefined') {
        delete window.__ODIGOS_SOURCE_PROFILER_TAB__;
      }
    };
  }, []);
  return null;
}
