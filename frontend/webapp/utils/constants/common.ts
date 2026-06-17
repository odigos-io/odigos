import { PlatformType, Tier } from '@odigos/ui-kit/types';
import type { OperationContext } from '@odigos/ui-kit/contexts/odigos-api';

export const IS_DEV = process.env.NODE_ENV === 'development';

const isLoopbackHost = typeof window !== 'undefined' ? /^(localhost|127\.0\.0\.1|\[::1\])$/.test(window.location.hostname) : false;

export const IS_LOCAL = IS_DEV && isLoopbackHost;

/**
 * Initial operation context used while we bootstrap. The real context
 * (platformType/tier/version) is derived by the layout via `useConfig`
 * and propagated through React rerenders.
 */
export const INITIAL_CONTEXT: OperationContext = {
  platformType: PlatformType.K8s,
  tier: Tier.Community,
  version: 'v0.0.0',
};
