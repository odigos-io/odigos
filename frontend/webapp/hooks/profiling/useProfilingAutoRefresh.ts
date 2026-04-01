import { useEffect } from 'react';
import type { FlamebearerProfile } from '@/types/profiling';

/**
 * Polls load() on an interval while the aggregated profile has no samples yet, so the UI picks up
 * data that arrives shortly after enable (OTLP is continuous; first GET is often empty).
 */
export function useProfilingAutoRefresh(
  load: (namespace: string, kind: string, name: string) => Promise<void>,
  namespace: string,
  kind: string,
  name: string,
  profile: FlamebearerProfile | null,
  options?: { enabled?: boolean; intervalMs?: number },
) {
  const enabled = options?.enabled ?? true;
  const intervalMs = options?.intervalMs ?? 4000;

  useEffect(() => {
    if (!enabled || !namespace?.trim() || !name?.trim() || !kind?.trim()) return;
    const ticks = profile?.flamebearer?.numTicks ?? 0;
    if (ticks > 0) return;

    const id = setInterval(() => {
      void load(namespace, kind, name);
    }, intervalMs);
    return () => clearInterval(id);
  }, [enabled, namespace, kind, name, load, profile?.flamebearer?.numTicks, intervalMs]);
}
