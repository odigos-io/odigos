/**
 * Mirrors InstrumentationRuleSpec.sourcesScopes (array of k8sconsts.SourcesScope JSON fields).
 * Kept separate from sampling.ts SourcesScope on purpose — duplication is intentional - since it doesn't have containerName
 */
export interface InstrumentationRuleSourcesScope {
  workloadName?: string | null;
  workloadKind?: string | null;
  workloadNamespace?: string | null;
  containerName?: string | null;
  workloadLanguage?: string | null;
}

export interface InstrumentationRuleSourcesScopeInput {
  workloadName?: string | null;
  workloadKind?: string | null;
  workloadNamespace?: string | null;
  containerName?: string | null;
  workloadLanguage?: string | null;
}
