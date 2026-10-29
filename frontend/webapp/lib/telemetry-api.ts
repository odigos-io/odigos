import type { TelemetryActivityDetail, TelemetryActivityPage, TelemetryApi } from '@odigos/ui-kit/types';

const SCOPE_FIELDS = 'serviceId service namespace operation';

const SUMMARY_FIELDS = `
  id
  updatedAt
  title
  before
  after
  targetSummary
  scopeSummary
  source { kind id name version }
  reason
  trigger {
    scope { ${SCOPE_FIELDS} }
    metric
    previousValue
    observedValue
    threshold
    sustainedSeconds
    observedAt
    requestCount
    measurement
    samples { at value }
  }
  status
  limitReason
`;

const GET_TELEMETRY_ACTIVITY = `
  query GetTelemetryActivity($filters: TelemetryActivityFiltersInput!, $cursor: String) {
    telemetryActivity(filters: $filters, cursor: $cursor) {
      items { ${SUMMARY_FIELDS} }
      nextCursor
      asOf
      historyLimit
    }
  }
`;

const GET_TELEMETRY_CHANGE = `
  query GetTelemetryChange($id: ID!) {
    telemetryChange(id: $id) {
      ${SUMMARY_FIELDS}
      targets { id scope { ${SCOPE_FIELDS} } before after broadScopeReason }
      timeline { id at title description }
      evidence { label value }
    }
  }
`;

const SAVE_HEAD_SAMPLING_RULE = `
  mutation SaveHeadSamplingRule($samplingId: ID!, $ruleId: ID, $rule: NoisyOperationRuleInput!, $surge: HeadSamplingSurgeInput, $surgeVersion: String!) {
    saveHeadSamplingRule(samplingId: $samplingId, ruleId: $ruleId, rule: $rule, surge: $surge, surgeVersion: $surgeVersion) {
      ruleId
    }
  }
`;

// GraphQL returns null for an absent optional field, the kit's types expect it left out.
const withoutNulls = <T>(value: T): T => {
  if (Array.isArray(value)) return value.map(withoutNulls) as T;
  if (value && typeof value === 'object') {
    return Object.fromEntries(
      Object.entries(value)
        .filter(([, v]) => v !== null)
        .map(([k, v]) => [k, withoutNulls(v)]),
    ) as T;
  }
  return value;
};

export interface TelemetryApiOptions {
  graphqlUrl: string;
  credentials: RequestCredentials;
  headers: () => Record<string, string>;
}

export const createTelemetryApi = ({ graphqlUrl, credentials, headers }: TelemetryApiOptions): TelemetryApi => {
  const request = async <T>(query: string, variables: Record<string, unknown>, signal: AbortSignal): Promise<T> => {
    const response = await fetch(graphqlUrl, {
      method: 'POST',
      credentials,
      headers: { 'Content-Type': 'application/json', ...headers() },
      body: JSON.stringify({ query, variables }),
      signal,
    });
    const body = await response.json().catch(() => undefined);
    if (body?.errors?.length) throw new Error(body.errors[0].message);
    if (!response.ok || !body?.data) throw new Error(`Telemetry request failed (${response.status}).`);
    return withoutNulls(body.data as T);
  };

  return {
    saveHeadSamplingRule: async ({ samplingId, ruleId, rule, surge, surgeVersion }, signal) => {
      await request(SAVE_HEAD_SAMPLING_RULE, { samplingId, ruleId: ruleId ?? null, rule, surge, surgeVersion }, signal);
    },
    getActivity: async (filters, cursor, signal) =>
      (await request<{ telemetryActivity: TelemetryActivityPage }>(GET_TELEMETRY_ACTIVITY, { filters, cursor: cursor ?? null }, signal)).telemetryActivity,
    getChange: async (id, signal) => (await request<{ telemetryChange: TelemetryActivityDetail }>(GET_TELEMETRY_CHANGE, { id }, signal)).telemetryChange,
  };
};
