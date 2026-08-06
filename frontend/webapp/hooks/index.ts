// Most CRUD/data hooks have been removed in favor of `useOdigosApi()` from
// `@odigos/ui-kit/contexts`. The hooks that remain in webapp are
// host-only concerns that don't fit the kit's API context model:
//   - auth/CSRF (`useCSRF`) — bootstraps the kit's Apollo client.
//   - long-lived listeners (`useSSE`, `useTokenTracker`) — drive multiple
//     stores from a side channel (SSE) and cross-cutting state.
//   - `useConfig` — Apollo `useSuspenseQuery` consumed by the layout to
//     derive platform/tier/version BEFORE the kit's API context is fully
//     wired (chicken-and-egg with the operation context).
//   - `useSetupHelpers` — orchestrates onboarding-step transitions, no API.
export * from './common';
export * from './config';
export * from './notification';
export * from './tokens';
