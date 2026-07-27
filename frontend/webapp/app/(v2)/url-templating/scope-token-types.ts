export type ScopeToken =
  | { type: 'entire_cluster' }
  | { type: 'namespace'; values: string[] }
  | { type: 'language'; values: string[] }
  | { type: 'workload'; groups: { kind: string; namespaceNames: string[] }[] };

export function scopeTokensSortKey(tokens: ScopeToken[]): string {
  return JSON.stringify(tokens);
}
