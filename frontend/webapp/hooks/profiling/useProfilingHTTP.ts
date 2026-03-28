'use client';

import { useCallback, useState } from 'react';
import { useMutation, useLazyQuery, gql } from '@apollo/client';
import type { FlamebearerProfile } from '@/types/profiling';
import { normalizeWorkloadKindForProfiling } from '@/utils/normalizeWorkloadKindForProfiling';
import { createCSRFHeaders, getCSRFTokenFromCookie } from '@/hooks/tokens/useCSRF';

const ENABLE_SOURCE_PROFILING = gql`
  mutation EnableSourceProfiling($namespace: String!, $kind: String!, $name: String!) {
    enableSourceProfiling(namespace: $namespace, kind: $kind, name: $name) {
      status
      sourceKey
      maxSlots
      activeSlots
    }
  }
`;

const GET_SOURCE_PROFILING = gql`
  query GetSourceProfiling($namespace: String!, $kind: String!, $name: String!) {
    sourceProfiling(namespace: $namespace, kind: $kind, name: $name) {
      profileJson
    }
  }
`;

/** GraphQL document for `profilingSlots` (diagnostics, polling). */
export const PROFILING_SLOTS_QUERY = gql`
  query GetProfilingSlots {
    profilingSlots {
      activeKeys
      keysWithData
      totalBytesUsed
      slotMaxBytes
      maxSlots
      maxTotalBytesBudget
      slotTtlSeconds
    }
  }
`;

export interface ProfilingSlotsDebug {
  activeKeys: string[];
  keysWithData: string[];
  totalBytesUsed: number;
  slotMaxBytes: number;
  maxSlots: number;
  maxTotalBytesBudget: number;
  slotTtlSeconds: number;
}

export interface EnableProfilingResult {
  status: string;
  sourceKey: string;
  maxSlots: number;
  activeSlots: number;
}

export interface UseProfilingHTTPState {
  loading: boolean;
  error: string | null;
  profile: FlamebearerProfile | null;
  lastSourceKey: string | null;
  enableMeta: EnableProfilingResult | null;
  load: (namespace: string, kind: string, name: string) => Promise<void>;
  enableAndLoad: (namespace: string, kind: string, name: string) => Promise<void>;
  clear: () => void;
}

export function useProfilingHTTP(): UseProfilingHTTPState {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [profile, setProfile] = useState<FlamebearerProfile | null>(null);
  const [lastSourceKey, setLastSourceKey] = useState<string | null>(null);
  const [enableMeta, setEnableMeta] = useState<EnableProfilingResult | null>(null);

  const [enableMutation] = useMutation<
    { enableSourceProfiling: EnableProfilingResult },
    { namespace: string; kind: string; name: string }
  >(ENABLE_SOURCE_PROFILING);

  const [queryProfiling] = useLazyQuery<
    { sourceProfiling: { profileJson: string } },
    { namespace: string; kind: string; name: string }
  >(GET_SOURCE_PROFILING, { fetchPolicy: 'no-cache' });

  const clear = useCallback(() => {
    setError(null);
    setProfile(null);
    setLastSourceKey(null);
    setEnableMeta(null);
  }, []);

  const load = useCallback(
    async (namespace: string, kind: string, name: string) => {
      const k = normalizeWorkloadKindForProfiling(kind);
      setLoading(true);
      setError(null);
      try {
        const { data, error: gqlError } = await queryProfiling({ variables: { namespace, kind: k, name } });
        if (gqlError) throw new Error(gqlError.message);
        if (!data) throw new Error('No data returned');
        const parsed = JSON.parse(data.sourceProfiling.profileJson) as FlamebearerProfile;
        setProfile(parsed);
        setLastSourceKey(`${namespace}/${k}/${name}`);
      } catch (e) {
        setProfile(null);
        setError(e instanceof Error ? e.message : String(e));
      } finally {
        setLoading(false);
      }
    },
    [queryProfiling],
  );

  const enableAndLoad = useCallback(
    async (namespace: string, kind: string, name: string) => {
      const k = normalizeWorkloadKindForProfiling(kind);
      setProfile(null);
      setEnableMeta(null);
      setLoading(true);
      setError(null);
      try {
        const { data: enData, errors: enErrors } = await enableMutation({ variables: { namespace, kind: k, name } });
        if (enErrors?.length) throw new Error(enErrors[0].message);
        if (enData) {
          setEnableMeta(enData.enableSourceProfiling);
          setLastSourceKey(enData.enableSourceProfiling.sourceKey);
        }
        const { data, error: gqlError } = await queryProfiling({ variables: { namespace, kind: k, name } });
        if (gqlError) throw new Error(gqlError.message);
        if (!data) throw new Error('No data returned');
        const parsed = JSON.parse(data.sourceProfiling.profileJson) as FlamebearerProfile;
        setProfile(parsed);
      } catch (e) {
        setProfile(null);
        setEnableMeta(null);
        setError(e instanceof Error ? e.message : String(e));
      } finally {
        setLoading(false);
      }
    },
    [enableMutation, queryProfiling],
  );

  return { loading, error, profile, lastSourceKey, enableMeta, load, enableAndLoad, clear };
}

/** Fetches active profiling slots via GraphQL (used outside React render in button handlers). */
export async function fetchProfilingSlotsDebug(): Promise<ProfilingSlotsDebug> {
  const query = PROFILING_SLOTS_QUERY.loc?.source.body ?? '';
  const { token } = getCSRFTokenFromCookie();
  const res = await fetch('/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...createCSRFHeaders(token) },
    body: JSON.stringify({ query }),
  });
  if (!res.ok) throw new Error(res.statusText);
  const json = (await res.json()) as { data?: { profilingSlots: ProfilingSlotsDebug }; errors?: { message: string }[] };
  if (json.errors?.length) throw new Error(json.errors[0].message);
  return json.data!.profilingSlots;
}
