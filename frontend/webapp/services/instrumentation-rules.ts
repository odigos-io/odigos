import { API } from '@/utils';
import { get, httpDelete, post, put } from './api';
import { InstrumentationRuleSpec } from '@/types'; // Ensure this type matches your backend data model

// API Endpoints
export async function getInstrumentationRules(): Promise<
  InstrumentationRuleSpec[]
> {
  return get(API.INSTRUMENTATION_RULES);
}

export async function getInstrumentationRule(
  id: string
): Promise<InstrumentationRuleSpec> {
  return get(API.INSTRUMENTATION_RULE(id));
}

export async function createInstrumentationRule(
  body: InstrumentationRuleSpec
): Promise<void> {
  return post(API.INSTRUMENTATION_RULES, body);
}

export async function updateInstrumentationRule(
  id: string,
  body: InstrumentationRuleSpec
): Promise<void> {
  return put(API.INSTRUMENTATION_RULE(id), body);
}

export async function deleteInstrumentationRule(id: string): Promise<void> {
  return httpDelete(API.INSTRUMENTATION_RULE(id));
}
