const env = process.env.NODE_ENV;

const LOCALHOST = 'http://localhost:8085/api';
const BASE_URL = env === 'production' ? '/api' : LOCALHOST;

const API = {
  EVENTS: `${BASE_URL}/events`,
  CONFIG: `${BASE_URL}/config`,
  NAMESPACES: `${BASE_URL}/namespaces`,
  APPLICATIONS: `${BASE_URL}/applications`,
  DESTINATION_TYPE: `${BASE_URL}/destination-types`,
  DESTINATIONS: `${BASE_URL}/destinations`,
  CHECK_CONNECTION: `${BASE_URL}/destinations/testConnection`,
  SOURCES: `${BASE_URL}/sources`,
  SET_ACTION: (type: string) => `${BASE_URL}/actions/types/${type}`,
  PUT_ACTION: (type: string, id: string) => `${BASE_URL}/actions/types/${type}/${id}`,
  ACTIONS: `${BASE_URL}/actions`,
  DELETE_ACTION: (type: string, id: string) => `${BASE_URL}/actions/types/${type}/${id}`,
  OVERVIEW_METRICS: `${BASE_URL}/metrics/overview`,
  INSTRUMENTATION_RULES: `${BASE_URL}/instrumentation-rules`,
  INSTRUMENTATION_RULE: (id: string) => `${BASE_URL}/instrumentation-rules/${id}`,
};

const QUERIES = {
  API_CONFIG: 'apiConfig',
  API_NAMESPACES: 'apiNamespaces',
  API_APPLICATIONS: 'apiApplications',
  API_DESTINATIONS: 'apiDestinations',
  API_SOURCES: 'apiSources',
  API_DESTINATION_TYPE: 'apiDestinationType',
  API_DESTINATION_TYPES: 'apiDestinationTypes',
  API_ACTIONS: 'apiActions',
};

const SLACK_INVITE_LINK = 'https://odigos.slack.com/join/shared_invite/zt-1d7egaz29-Rwv2T8kyzc3mWP8qKobz~A#/shared-invite/email';

export const DOCS_LINK = 'https://docs.odigos.io';
export const ACTION_DOCS_LINK = `${DOCS_LINK}/pipeline/actions/introduction`;
export const ACTION_ITEM_DOCS_LINK = `${DOCS_LINK}/pipeline/actions`;
export const INSTRUMENTATION_RULES_DOCS_LINK = `${DOCS_LINK}/pipeline/rules/overview`;

export { API, QUERIES, SLACK_INVITE_LINK, BASE_URL };
